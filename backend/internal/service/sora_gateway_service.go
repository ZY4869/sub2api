package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

// SoraGatewayService handles forwarding requests to Sora upstream.
type SoraGatewayService struct {
	soraClient       SoraClient
	rateLimitService *RateLimitService
	httpUpstream     HTTPUpstream // 用于 apikey 类型账号的 HTTP 透传
	cfg              *config.Config
}

type soraPreflightChecker interface {
	PreflightCheck(ctx context.Context, account *Account, requestedModel string, modelCfg SoraModelConfig) error
}

func NewSoraGatewayService(
	soraClient SoraClient,
	rateLimitService *RateLimitService,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
) *SoraGatewayService {
	return &SoraGatewayService{
		soraClient:       soraClient,
		rateLimitService: rateLimitService,
		httpUpstream:     httpUpstream,
		cfg:              cfg,
	}
}

func (s *SoraGatewayService) Forward(ctx context.Context, c *gin.Context, account *Account, body []byte, clientStream bool) (*ForwardResult, error) {
	startTime := time.Now()

	// apikey 类型账号：HTTP 透传到上游，不走 SoraSDKClient
	if account.Type == AccountTypeAPIKey && account.GetBaseURL() != "" {
		if s.httpUpstream == nil {
			s.writeSoraError(c, http.StatusInternalServerError, "api_error", "HTTP upstream client not configured", clientStream)
			return nil, errors.New("httpUpstream not configured for sora apikey forwarding")
		}
		return s.forwardToUpstream(ctx, c, account, body, clientStream, startTime)
	}

	if s.soraClient == nil || !s.soraClient.Enabled() {
		if c != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"type":    "api_error",
					"message": "Sora 上游未配置",
				},
			})
		}
		return nil, errors.New("sora upstream not configured")
	}

	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		s.writeSoraError(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body", clientStream)
		return nil, fmt.Errorf("parse request: %w", err)
	}
	reqModel, _ := reqBody["model"].(string)
	reqStream, _ := reqBody["stream"].(bool)
	if strings.TrimSpace(reqModel) == "" {
		s.writeSoraError(c, http.StatusBadRequest, "invalid_request_error", "model is required", clientStream)
		return nil, errors.New("model is required")
	}

	mappedModel := account.GetMappedModel(reqModel)
	if mappedModel != "" && mappedModel != reqModel {
		reqModel = mappedModel
	}

	modelCfg, ok := GetSoraModelConfig(reqModel)
	if !ok {
		s.writeSoraError(c, http.StatusBadRequest, "invalid_request_error", "Unsupported Sora model", clientStream)
		return nil, fmt.Errorf("unsupported model: %s", reqModel)
	}
	prompt, imageInput, videoInput, remixTargetID := extractSoraInput(reqBody)
	prompt = strings.TrimSpace(prompt)
	imageInput = strings.TrimSpace(imageInput)
	videoInput = strings.TrimSpace(videoInput)
	remixTargetID = strings.TrimSpace(remixTargetID)

	if videoInput != "" && modelCfg.Type != "video" {
		s.writeSoraError(c, http.StatusBadRequest, "invalid_request_error", "video input only supports video models", clientStream)
		return nil, errors.New("video input only supports video models")
	}
	if videoInput != "" && imageInput != "" {
		s.writeSoraError(c, http.StatusBadRequest, "invalid_request_error", "image input and video input cannot be used together", clientStream)
		return nil, errors.New("image input and video input cannot be used together")
	}
	characterOnly := videoInput != "" && prompt == ""
	if modelCfg.Type == "prompt_enhance" && prompt == "" {
		s.writeSoraError(c, http.StatusBadRequest, "invalid_request_error", "prompt is required", clientStream)
		return nil, errors.New("prompt is required")
	}
	if modelCfg.Type != "prompt_enhance" && prompt == "" && !characterOnly {
		s.writeSoraError(c, http.StatusBadRequest, "invalid_request_error", "prompt is required", clientStream)
		return nil, errors.New("prompt is required")
	}

	reqCtx, cancel := s.withSoraTimeout(ctx, reqStream)
	if cancel != nil {
		defer cancel()
	}
	if checker, ok := s.soraClient.(soraPreflightChecker); ok && !characterOnly {
		if err := checker.PreflightCheck(reqCtx, account, reqModel, modelCfg); err != nil {
			return nil, s.handleSoraRequestError(ctx, account, err, reqModel, c, clientStream)
		}
	}

	if modelCfg.Type == "prompt_enhance" {
		enhancedPrompt, err := s.soraClient.EnhancePrompt(reqCtx, account, prompt, modelCfg.ExpansionLevel, modelCfg.DurationS)
		if err != nil {
			return nil, s.handleSoraRequestError(ctx, account, err, reqModel, c, clientStream)
		}
		content := strings.TrimSpace(enhancedPrompt)
		if content == "" {
			content = prompt
		}
		var firstTokenMs *int
		if clientStream {
			ms, streamErr := s.writeSoraStream(c, reqModel, content, startTime)
			if streamErr != nil {
				return nil, streamErr
			}
			firstTokenMs = ms
		} else if c != nil {
			c.JSON(http.StatusOK, buildSoraNonStreamResponse(content, reqModel))
		}
		return &ForwardResult{
			RequestID:    "",
			Model:        reqModel,
			Stream:       clientStream,
			Duration:     time.Since(startTime),
			FirstTokenMs: firstTokenMs,
			Usage:        ClaudeUsage{},
			MediaType:    "prompt",
		}, nil
	}

	characterOpts := parseSoraCharacterOptions(reqBody)
	watermarkOpts := parseSoraWatermarkOptions(reqBody)
	var characterResult *soraCharacterFlowResult
	if videoInput != "" {
		videoData, videoErr := decodeSoraVideoInput(reqCtx, videoInput)
		if videoErr != nil {
			s.writeSoraError(c, http.StatusBadRequest, "invalid_request_error", videoErr.Error(), clientStream)
			return nil, videoErr
		}
		characterResult, videoErr = s.createCharacterFromVideo(reqCtx, account, videoData, characterOpts)
		if videoErr != nil {
			return nil, s.handleSoraRequestError(ctx, account, videoErr, reqModel, c, clientStream)
		}
		if characterResult != nil && characterOpts.DeleteAfterGenerate && strings.TrimSpace(characterResult.CharacterID) != "" && !characterOnly {
			characterID := strings.TrimSpace(characterResult.CharacterID)
			defer func() {
				cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancelCleanup()
				if err := s.soraClient.DeleteCharacter(cleanupCtx, account, characterID); err != nil {
					log.Printf("[Sora] cleanup character failed, character_id=%s err=%v", characterID, err)
				}
			}()
		}
		if characterOnly {
			content := "角色创建成功"
			if characterResult != nil && strings.TrimSpace(characterResult.Username) != "" {
				content = fmt.Sprintf("角色创建成功，角色名@%s", strings.TrimSpace(characterResult.Username))
			}
			var firstTokenMs *int
			if clientStream {
				ms, streamErr := s.writeSoraStream(c, reqModel, content, startTime)
				if streamErr != nil {
					return nil, streamErr
				}
				firstTokenMs = ms
			} else if c != nil {
				resp := buildSoraNonStreamResponse(content, reqModel)
				if characterResult != nil {
					resp["character_id"] = characterResult.CharacterID
					resp["cameo_id"] = characterResult.CameoID
					resp["character_username"] = characterResult.Username
					resp["character_display_name"] = characterResult.DisplayName
				}
				c.JSON(http.StatusOK, resp)
			}
			return &ForwardResult{
				RequestID:    "",
				Model:        reqModel,
				Stream:       clientStream,
				Duration:     time.Since(startTime),
				FirstTokenMs: firstTokenMs,
				Usage:        ClaudeUsage{},
				MediaType:    "prompt",
			}, nil
		}
		if characterResult != nil && strings.TrimSpace(characterResult.Username) != "" {
			prompt = fmt.Sprintf("@%s %s", characterResult.Username, prompt)
		}
	}

	var imageData []byte
	imageFilename := ""
	if imageInput != "" {
		decoded, filename, err := decodeSoraImageInput(reqCtx, imageInput)
		if err != nil {
			s.writeSoraError(c, http.StatusBadRequest, "invalid_request_error", err.Error(), clientStream)
			return nil, err
		}
		imageData = decoded
		imageFilename = filename
	}

	mediaID := ""
	if len(imageData) > 0 {
		uploadID, err := s.soraClient.UploadImage(reqCtx, account, imageData, imageFilename)
		if err != nil {
			return nil, s.handleSoraRequestError(ctx, account, err, reqModel, c, clientStream)
		}
		mediaID = uploadID
	}

	taskID := ""
	var err error
	videoCount := parseSoraVideoCount(reqBody)
	switch modelCfg.Type {
	case "image":
		taskID, err = s.soraClient.CreateImageTask(reqCtx, account, SoraImageRequest{
			Prompt:  prompt,
			Width:   modelCfg.Width,
			Height:  modelCfg.Height,
			MediaID: mediaID,
		})
	case "video":
		if remixTargetID == "" && isSoraStoryboardPrompt(prompt) {
			taskID, err = s.soraClient.CreateStoryboardTask(reqCtx, account, SoraStoryboardRequest{
				Prompt:      formatSoraStoryboardPrompt(prompt),
				Orientation: modelCfg.Orientation,
				Frames:      modelCfg.Frames,
				Model:       modelCfg.Model,
				Size:        modelCfg.Size,
				MediaID:     mediaID,
			})
		} else {
			taskID, err = s.soraClient.CreateVideoTask(reqCtx, account, SoraVideoRequest{
				Prompt:        prompt,
				Orientation:   modelCfg.Orientation,
				Frames:        modelCfg.Frames,
				Model:         modelCfg.Model,
				Size:          modelCfg.Size,
				VideoCount:    videoCount,
				MediaID:       mediaID,
				RemixTargetID: remixTargetID,
				CameoIDs:      extractSoraCameoIDs(reqBody),
			})
		}
	default:
		err = fmt.Errorf("unsupported model type: %s", modelCfg.Type)
	}
	if err != nil {
		return nil, s.handleSoraRequestError(ctx, account, err, reqModel, c, clientStream)
	}

	if clientStream && c != nil {
		s.prepareSoraStream(c, taskID)
	}

	var mediaURLs []string
	videoGenerationID := ""
	mediaType := modelCfg.Type
	imageCount := 0
	imageSize := ""
	switch modelCfg.Type {
	case "image":
		urls, pollErr := s.pollImageTask(reqCtx, c, account, taskID, clientStream)
		if pollErr != nil {
			return nil, s.handleSoraRequestError(ctx, account, pollErr, reqModel, c, clientStream)
		}
		mediaURLs = urls
		imageCount = len(urls)
		imageSize = soraImageSizeFromModel(reqModel)
	case "video":
		videoStatus, pollErr := s.pollVideoTaskDetailed(reqCtx, c, account, taskID, clientStream)
		if pollErr != nil {
			return nil, s.handleSoraRequestError(ctx, account, pollErr, reqModel, c, clientStream)
		}
		if videoStatus != nil {
			mediaURLs = videoStatus.URLs
			videoGenerationID = strings.TrimSpace(videoStatus.GenerationID)
		}
	default:
		mediaType = "prompt"
	}

	watermarkPostID := ""
	if modelCfg.Type == "video" && watermarkOpts.Enabled {
		watermarkURL, postID, watermarkErr := s.resolveWatermarkFreeURL(reqCtx, account, videoGenerationID, watermarkOpts)
		if watermarkErr != nil {
			if !watermarkOpts.FallbackOnFailure {
				return nil, s.handleSoraRequestError(ctx, account, watermarkErr, reqModel, c, clientStream)
			}
			log.Printf("[Sora] watermark-free fallback to original URL, task_id=%s err=%v", taskID, watermarkErr)
		} else if strings.TrimSpace(watermarkURL) != "" {
			mediaURLs = []string{strings.TrimSpace(watermarkURL)}
			watermarkPostID = strings.TrimSpace(postID)
		}
	}

	// 直调路径（/sora/v1/chat/completions）保持纯透传，不执行本地/S3 媒体落盘。
	// 媒体存储由客户端 API 路径（/api/v1/sora/generate）的异步流程负责。
	finalURLs := s.normalizeSoraMediaURLs(mediaURLs)
	if watermarkPostID != "" && watermarkOpts.DeletePost {
		if deleteErr := s.soraClient.DeletePost(reqCtx, account, watermarkPostID); deleteErr != nil {
			log.Printf("[Sora] delete post failed, post_id=%s err=%v", watermarkPostID, deleteErr)
		}
	}

	content := buildSoraContent(mediaType, finalURLs)
	var firstTokenMs *int
	if clientStream {
		ms, streamErr := s.writeSoraStream(c, reqModel, content, startTime)
		if streamErr != nil {
			return nil, streamErr
		}
		firstTokenMs = ms
	} else if c != nil {
		response := buildSoraNonStreamResponse(content, reqModel)
		if len(finalURLs) > 0 {
			response["media_url"] = finalURLs[0]
			if len(finalURLs) > 1 {
				response["media_urls"] = finalURLs
			}
		}
		c.JSON(http.StatusOK, response)
	}

	return &ForwardResult{
		RequestID:    taskID,
		Model:        reqModel,
		Stream:       clientStream,
		Duration:     time.Since(startTime),
		FirstTokenMs: firstTokenMs,
		Usage:        ClaudeUsage{},
		MediaType:    mediaType,
		MediaURL:     firstMediaURL(finalURLs),
		ImageCount:   imageCount,
		ImageSize:    imageSize,
	}, nil
}

func (s *SoraGatewayService) withSoraTimeout(ctx context.Context, stream bool) (context.Context, context.CancelFunc) {
	if s == nil || s.cfg == nil {
		return ctx, nil
	}
	timeoutSeconds := s.cfg.Gateway.SoraRequestTimeoutSeconds
	if stream {
		timeoutSeconds = s.cfg.Gateway.SoraStreamTimeoutSeconds
	}
	if timeoutSeconds <= 0 {
		return ctx, nil
	}
	return context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
}
