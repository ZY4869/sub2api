package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

type soraCharacterFlowResult struct {
	CameoID     string
	CharacterID string
	Username    string
	DisplayName string
}

func (s *SoraGatewayService) createCharacterFromVideo(ctx context.Context, account *Account, videoData []byte, opts soraCharacterOptions) (*soraCharacterFlowResult, error) {
	cameoID, err := s.soraClient.UploadCharacterVideo(ctx, account, videoData)
	if err != nil {
		return nil, err
	}

	cameoStatus, err := s.pollCameoStatus(ctx, account, cameoID)
	if err != nil {
		return nil, err
	}
	username := processSoraCharacterUsername(cameoStatus.UsernameHint)
	displayName := strings.TrimSpace(cameoStatus.DisplayNameHint)
	if displayName == "" {
		displayName = "Character"
	}
	profileAssetURL := strings.TrimSpace(cameoStatus.ProfileAssetURL)
	if profileAssetURL == "" {
		return nil, errors.New("profile asset url not found in cameo status")
	}

	avatarData, err := s.soraClient.DownloadCharacterImage(ctx, account, profileAssetURL)
	if err != nil {
		return nil, err
	}
	assetPointer, err := s.soraClient.UploadCharacterImage(ctx, account, avatarData)
	if err != nil {
		return nil, err
	}
	instructionSet := cameoStatus.InstructionSetHint
	if instructionSet == nil {
		instructionSet = cameoStatus.InstructionSet
	}

	characterID, err := s.soraClient.FinalizeCharacter(ctx, account, SoraCharacterFinalizeRequest{
		CameoID:             strings.TrimSpace(cameoID),
		Username:            username,
		DisplayName:         displayName,
		ProfileAssetPointer: assetPointer,
		InstructionSet:      instructionSet,
	})
	if err != nil {
		return nil, err
	}

	if opts.SetPublic {
		if err := s.soraClient.SetCharacterPublic(ctx, account, cameoID); err != nil {
			return nil, err
		}
	}

	return &soraCharacterFlowResult{
		CameoID:     strings.TrimSpace(cameoID),
		CharacterID: strings.TrimSpace(characterID),
		Username:    strings.TrimSpace(username),
		DisplayName: displayName,
	}, nil
}

func (s *SoraGatewayService) pollCameoStatus(ctx context.Context, account *Account, cameoID string) (*SoraCameoStatus, error) {
	timeout := 10 * time.Minute
	interval := 5 * time.Second
	maxAttempts := int(math.Ceil(timeout.Seconds() / interval.Seconds()))
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	consecutiveErrors := 0
	for attempt := 0; attempt < maxAttempts; attempt++ {
		status, err := s.soraClient.GetCameoStatus(ctx, account, cameoID)
		if err != nil {
			lastErr = err
			consecutiveErrors++
			if consecutiveErrors >= 3 {
				break
			}
			if attempt < maxAttempts-1 {
				if sleepErr := sleepWithContext(ctx, interval); sleepErr != nil {
					return nil, sleepErr
				}
			}
			continue
		}
		consecutiveErrors = 0
		if status == nil {
			if attempt < maxAttempts-1 {
				if sleepErr := sleepWithContext(ctx, interval); sleepErr != nil {
					return nil, sleepErr
				}
			}
			continue
		}
		currentStatus := strings.ToLower(strings.TrimSpace(status.Status))
		statusMessage := strings.TrimSpace(status.StatusMessage)
		if currentStatus == "failed" {
			if statusMessage == "" {
				statusMessage = "character creation failed"
			}
			return nil, errors.New(statusMessage)
		}
		if strings.EqualFold(statusMessage, "Completed") || currentStatus == "finalized" {
			return status, nil
		}
		if attempt < maxAttempts-1 {
			if sleepErr := sleepWithContext(ctx, interval); sleepErr != nil {
				return nil, sleepErr
			}
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("poll cameo status failed: %w", lastErr)
	}
	return nil, errors.New("cameo processing timeout")
}

func processSoraCharacterUsername(usernameHint string) string {
	usernameHint = strings.TrimSpace(usernameHint)
	if usernameHint == "" {
		usernameHint = "character"
	}
	if strings.Contains(usernameHint, ".") {
		parts := strings.Split(usernameHint, ".")
		usernameHint = strings.TrimSpace(parts[len(parts)-1])
	}
	if usernameHint == "" {
		usernameHint = "character"
	}
	return fmt.Sprintf("%s%d", usernameHint, rand.Intn(900)+100)
}

func (s *SoraGatewayService) resolveWatermarkFreeURL(ctx context.Context, account *Account, generationID string, opts soraWatermarkOptions) (string, string, error) {
	generationID = strings.TrimSpace(generationID)
	if generationID == "" {
		return "", "", errors.New("generation id is required for watermark-free mode")
	}
	postID, err := s.soraClient.PostVideoForWatermarkFree(ctx, account, generationID)
	if err != nil {
		return "", "", err
	}
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return "", "", errors.New("watermark-free publish returned empty post id")
	}

	switch opts.ParseMethod {
	case "custom":
		urlVal, parseErr := s.soraClient.GetWatermarkFreeURLCustom(ctx, account, opts.ParseURL, opts.ParseToken, postID)
		if parseErr != nil {
			return "", postID, parseErr
		}
		return strings.TrimSpace(urlVal), postID, nil
	case "", "third_party":
		return fmt.Sprintf("https://oscdn2.dyysy.com/MP4/%s.mp4", postID), postID, nil
	default:
		return "", postID, fmt.Errorf("unsupported watermark parse method: %s", opts.ParseMethod)
	}
}
