package internal

import (
	"log"
	"path/filepath"
)

const (
	pwc = '*' // placeholder wild character
	sep = '.'
)

// TopicHasWildcard checks if the topic is a wildcard.
func TopicHasWildcard(topic string) bool {
	for i, c := range topic {
		if c == pwc {
			if (i == 0 || topic[i-1] == sep) &&
				(i+1 == len(topic) || topic[i+1] == sep) {
				return true
			}
		}
	}

	return false
}

// FindTopicsList find topics that match the topic pattern.
func FindTopicsList(topics []string, pattern string) []string {
	var matchedTopics []string

	for _, topic := range topics {
		ok, err := filepath.Match(pattern, topic)
		if ok {
			matchedTopics = append(matchedTopics, topic)
		} else if err != nil {
			log.Println("error in topic matching")
		}
	}

	return matchedTopics
}

// FindRelatedWildcardTopics find topics patterns that are applicable to the given topic.
func FindRelatedWildcardTopics(topic string, topics []string) []string {
	var matchedTopics []string

	for _, pattern := range topics {
		ok, err := filepath.Match(pattern, topic)
		if ok {
			matchedTopics = append(matchedTopics, pattern)
		} else if err != nil {
			log.Println("error in topic matching")
		}
	}

	return matchedTopics
}

// IsSubscribeTopicValid check the subscribed topic exist or matched with client topics.
func IsSubscribeTopicValid(topic string, topics []string) bool {
	for _, t := range topics {
		ok, _ := filepath.Match(topic, t)
		if ok {
			return true
		}
	}

	return false
}

// AppendIfMissing check if item missing append item to list.
func AppendIfMissing(topics []string, topic string) []string {
	for _, item := range topics {
		if item == topic {
			return topics
		}
	}

	return append(topics, topic)
}
