package cli

import (
	"context"
	"fmt"
	"slices"
)

type ghAPIGetArrayFunc func(context.Context, string, string) ([]map[string]any, error)

func countHumanComments(comments []map[string]any) int {
	count := 0
	for _, comment := range comments {
		if isHumanComment(comment) {
			count++
		}
	}
	return count
}

func countHumanCommentsAfter(comments []map[string]any, createdAt string) int {
	count := 0
	for _, comment := range comments {
		commentCreatedAt, _ := comment["created_at"].(string)
		if commentCreatedAt > createdAt && isHumanComment(comment) {
			count++
		}
	}
	return count
}

func isHumanComment(comment map[string]any) bool {
	user, _ := comment["user"].(map[string]any)
	login, _ := user["login"].(string)
	return !isBotUser(login)
}

func isLatestCloseByBot(ctx context.Context, number int, repo string, getEvents ghAPIGetArrayFunc) (bool, error) {
	events, err := getEvents(ctx, fmt.Sprintf("issues/%d/events", number), repo)
	if err != nil {
		return false, err
	}
	for i := range slices.Backward(events) {
		event, _ := events[i]["event"].(string)
		if event != "closed" {
			continue
		}
		actor, _ := events[i]["actor"].(map[string]any)
		login, _ := actor["login"].(string)
		return isBotUser(login), nil
	}
	return false, fmt.Errorf("no close event found for %s#%d", repo, number)
}

func labelNamesFromMaps(count int, labelAt func(int) map[string]any) []string {
	if count == 0 {
		return []string{}
	}
	result := make([]string, 0, count)
	for i := range count {
		labelMap := labelAt(i)
		if name, ok := labelMap["name"].(string); ok {
			result = append(result, name)
		}
	}
	return result
}
