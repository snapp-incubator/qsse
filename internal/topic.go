package internal

import (
	"log"
	"path/filepath"
)

const (
	pwc = '*' //placeholder wild character
	sep = '.'
)

//topicHasWildcard checks if the topic is a wildcard.
func topicHasWildcard(topic string) bool {
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

//findTopicsList find topics that match the topic pattern.
func findTopicsList(topics []string, pattern string) []string {
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

//findRelatedWildcardTopics find topics patterns that are applicable to the given topic.
func findRelatedWildcardTopics(topic string, topics []string) []string {
	var matchedTopics []string
	for _, pattern := range topics {
		if topicHasWildcard(pattern) {
			ok, err := filepath.Match(pattern, topic)
			if ok {
				matchedTopics = append(matchedTopics, pattern)
			} else if err != nil {
				log.Println("error in topic matching")
			}
		}
	}
	return matchedTopics
}
