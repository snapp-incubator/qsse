package internal_test

import (
	"testing"

	"github.com/snapp-incubator/qsse/internal"
	"github.com/stretchr/testify/assert"
)

func TestTopicHasWildcard(t *testing.T) {
	tests := []struct {
		name        string
		topic       string
		hasWildcard bool
	}{
		{
			name:        "topic has wildcard",
			topic:       "ride.passenger.*",
			hasWildcard: true,
		},
		{
			name:        "topic has wildcard",
			topic:       "ride.*.start",
			hasWildcard: true,
		},
		{
			name:        "topic has wildcard",
			topic:       "*.ride.*",
			hasWildcard: true,
		},
		{
			name:        "topic has not  wildcard",
			topic:       "ride.passenger.125",
			hasWildcard: false,
		},
	}

	for _, test := range tests {
		testCase := test
		t.Run(test.name, func(t *testing.T) {
			result := internal.TopicHasWildcard(testCase.topic)
			assert.Equal(t, testCase.hasWildcard, result)
		})
	}
}

func TestFindTopicsList(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		topics        []string
		matchedTopics []string
	}{
		{
			name:          "empty result",
			pattern:       "ride.*",
			topics:        []string{"passenger.online", "driver.online"},
			matchedTopics: []string{},
		},
		{
			name:          "has matched topics",
			pattern:       "ride.*",
			topics:        []string{"ride.accepted", "ride.rejected", "ride.finished", "offer.first"},
			matchedTopics: []string{"ride.accepted", "ride.rejected", "ride.finished"},
		},
		{
			name:          "has matched topics",
			pattern:       "ride.accepted",
			topics:        []string{"ride.accepted", "ride.rejected", "ride.finished", "offer.first"},
			matchedTopics: []string{"ride.accepted"},
		},
	}

	f := internal.Finder{
		Logger: internal.NewLogger().Named("topic-finder"),
	}

	for _, test := range tests {
		testCase := test
		t.Run(test.name, func(t *testing.T) {
			result := f.FindTopicsList(testCase.topics, testCase.pattern)
			assert.Equal(t, len(testCase.matchedTopics), len(result))
		})

	}
}

func TestFindRelatedWildcardTopics(t *testing.T) {
	tests := []struct {
		name          string
		topic         string
		topics        []string
		matchedTopics []string
	}{
		{
			name:          "empty result",
			topic:         "call.*",
			topics:        []string{"ride.*", "call.start", "ride.driver.*"},
			matchedTopics: []string{},
		},
		{
			name:          "has matched topic",
			topic:         "ride.start",
			topics:        []string{"ride.*", "call.start"},
			matchedTopics: []string{"ride.*"},
		},
		{
			name:          "has multiple matched topic",
			topic:         "ride.driver.*",
			topics:        []string{"ride.*", "call.start", "ride.driver.*"},
			matchedTopics: []string{"ride.*", "ride.driver.*"},
		},
	}

	f := internal.Finder{
		Logger: internal.NewLogger().Named("topic-finder"),
	}

	for _, test := range tests {
		testCase := test
		t.Run(test.name, func(t *testing.T) {
			result := f.FindRelatedWildcardTopics(
				testCase.topic,
				testCase.topics,
			)
			assert.Equal(t, len(testCase.matchedTopics), len(result))
		})
	}
}
