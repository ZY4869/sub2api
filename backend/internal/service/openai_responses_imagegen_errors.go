package service

import (
	"errors"
	"strings"
)

func newOpenAIResponsesCompatError(status int, errType string, code string, message string) *OpenAIResponsesCompatError {
	return newOpenAIResponsesCompatErrorWithMetadata(status, errType, code, message, OpenAIResponsesCompatMetadata{})
}

func newOpenAIResponsesCompatErrorWithMetadata(status int, errType string, code string, message string, metadata OpenAIResponsesCompatMetadata) *OpenAIResponsesCompatError {
	if metadata.Rejected {
		metadata.Enabled = false
		if strings.TrimSpace(metadata.RejectCode) == "" {
			metadata.RejectCode = strings.TrimSpace(code)
		}
	}
	return &OpenAIResponsesCompatError{
		Status:   status,
		Type:     strings.TrimSpace(errType),
		Code:     strings.TrimSpace(code),
		Message:  strings.TrimSpace(message),
		Metadata: metadata,
	}
}

func enrichOpenAIResponsesCompatRejectMetadata(target *OpenAIResponsesCompatError, metadata OpenAIResponsesCompatMetadata) {
	if target == nil {
		return
	}
	target.Metadata = mergeOpenAIResponsesCompatMetadata(target.Metadata, metadata)
	if target.Metadata.Rejected && strings.TrimSpace(target.Metadata.RejectCode) == "" {
		target.Metadata.RejectCode = strings.TrimSpace(target.Code)
	}
}

func withOpenAIResponsesCompatRejectMetadata(err error, metadata OpenAIResponsesCompatMetadata, referenceImageCount int) error {
	var compatErr *OpenAIResponsesCompatError
	if !errors.As(err, &compatErr) {
		return err
	}
	metadata.Rejected = true
	metadata.Enabled = false
	metadata.ReferenceImagesNormalized = false
	if referenceImageCount > metadata.ReferenceImageCount {
		metadata.ReferenceImageCount = referenceImageCount
	}
	enrichOpenAIResponsesCompatRejectMetadata(compatErr, metadata)
	return compatErr
}

func mergeOpenAIResponsesCompatMetadata(current OpenAIResponsesCompatMetadata, updates OpenAIResponsesCompatMetadata) OpenAIResponsesCompatMetadata {
	if current.Source == "" {
		current.Source = strings.TrimSpace(updates.Source)
	}
	if current.SourceGuess == "" {
		current.SourceGuess = strings.TrimSpace(updates.SourceGuess)
	}
	if updates.Rejected {
		current.Rejected = true
	}
	if current.RejectCode == "" {
		current.RejectCode = strings.TrimSpace(updates.RejectCode)
	}
	if updates.ReferenceImageCount > current.ReferenceImageCount {
		current.ReferenceImageCount = updates.ReferenceImageCount
	}
	if updates.ReferenceImageBytesBefore > current.ReferenceImageBytesBefore {
		current.ReferenceImageBytesBefore = updates.ReferenceImageBytesBefore
	}
	if updates.ReferenceImageBytesAfter > current.ReferenceImageBytesAfter {
		current.ReferenceImageBytesAfter = updates.ReferenceImageBytesAfter
	}
	if updates.ReferenceImagesNormalized {
		current.ReferenceImagesNormalized = true
	}
	if current.ImageGenerationSize == "" {
		current.ImageGenerationSize = strings.TrimSpace(updates.ImageGenerationSize)
	}
	return current
}

func maxOpenAIResponsesCompatReferenceImageCount(values ...int) int {
	maxValue := 0
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}
