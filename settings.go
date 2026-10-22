package files

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const (
	SettingPhotoLongestSide  = "photo_longest_side"
	SettingPhotosPerQuestion = "photos_per_question"
	SettingLargestFileBytes  = "largest_file_bytes"
)

type SettingsLayers func(ctx context.Context) (definition map[string]any, overrides map[string]spec.Override, err error)

type Settings struct {
	PhotoLongestSide  int              `json:"photo_longest_side"`
	PhotosPerQuestion int              `json:"photos_per_question"`
	LargestFileBytes  int64            `json:"largest_file_bytes"`
	Effective         []spec.Effective `json:"effective"`
}

func (m *fileManager) UseSettings(layers SettingsLayers) { m.settings = layers }

func (m *fileManager) Settings(ctx context.Context) (Settings, error) {
	var definition map[string]any
	var overrides map[string]spec.Override
	if m.settings != nil {
		var err error
		definition, overrides, err = m.settings(ctx)
		if err != nil {
			return Settings{}, fmt.Errorf("Settings: %w", err)
		}
	}
	out := Settings{Effective: spec.ResolveAll(m.spec.Settings, definition, overrides)}
	for _, e := range out.Effective {
		n, ok := wholeNumber(e.Value)
		if !ok {
			return Settings{}, fmt.Errorf("Settings: %q resolved to %v, which is not a whole number", e.Name, e.Value)
		}
		switch e.Name {
		case SettingPhotoLongestSide:
			out.PhotoLongestSide = int(n)
		case SettingPhotosPerQuestion:
			out.PhotosPerQuestion = int(n)
		case SettingLargestFileBytes:
			out.LargestFileBytes = n
		}
	}
	return out, nil
}

func (m *fileManager) fitsLargestFile(ctx context.Context, size int) error {
	s, err := m.Settings(ctx)
	if err != nil {
		return err
	}
	if int64(size) > s.LargestFileBytes {
		return fmt.Errorf("%w: %d bytes, the largest is %d", ErrFileTooLarge, size, s.LargestFileBytes)
	}
	return nil
}

func wholeNumber(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), n == float64(int64(n))
	case float32:
		return int64(n), n == float32(int64(n))
	}
	return 0, false
}
