package blogservice

import "time"

func normalizePublicationState(published bool, publishAt *time.Time, now time.Time) (bool, *time.Time, *time.Time) {
	var normalizedPublishAt *time.Time

	if publishAt != nil {
		value := publishAt.UTC()
		normalizedPublishAt = &value
	}

	if !published {
		return false, normalizedPublishAt, nil
	}

	if normalizedPublishAt == nil {
		value := now.UTC()
		normalizedPublishAt = &value
	} else {
		value := normalizedPublishAt.UTC()
		normalizedPublishAt = &value
	}

	publishedAtValue := normalizedPublishAt.UTC()
	return true, normalizedPublishAt, &publishedAtValue
}
