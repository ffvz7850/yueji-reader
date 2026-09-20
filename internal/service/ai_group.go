package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// ============================================================
// Phase 6-1: AI 智能分组检测增强
// ============================================================

// AIGroupCandidate 一组候选漫画，供 AI 分析是否属于同系列
type AIGroupCandidate struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// AIGroupSuggestion AI 分析后返回的分组建议
type AIGroupSuggestion struct {
	GroupName string   `json:"groupName"` // 建议的分组名
	ComicIDs  []string `json:"comicIds"`  // 属于该组的漫画 ID
	Reason    string   `json:"reason"`    // AI 判断理由
}

// AIAnalyzeGroupCandidates 使用 AI 对候选漫画进行语义分析，判断哪些属于同系列。
// candidates: 所有未分组的漫画 ID+标题
// 返回 AI 建议的分组列表
func AIAnalyzeGroupCandidates(cfg AIConfig, candidates []AIGroupCandidate, targetLang string) ([]AIGroupSuggestion, error) {
	if !cfg.EnableCloudAI || cfg.CloudAPIKey == "" {
		return nil, fmt.Errorf("cloud AI not configured")
	}

	langName := "Chinese (简体中文)"
	if targetLang == "en" {
		langName = "English"
	}

	systemPrompt := fmt.Sprintf(`You are an expert manga/comic librarian. Given a list of comic titles, group them by series.

Rules:
- Identify comics that belong to the SAME series (same work, different volumes/editions)
- Consider: different volume numbers, different translations of the same title, different editions, sequel/prequel with VERY similar names
- Do NOT group different works together just because they share a genre or author
- Handle multilingual titles: "進撃の巨人" = "Attack on Titan" = "进击的巨人"
- Handle common filename patterns: "[Group] Title Vol.01", "Title 第1巻", "Title_v01"
- Each group must have at least 2 comics
- Provide a clean series name for each group (without volume numbers) in %s
- Return ONLY a JSON array, no extra text or markdown

Response format:
[
  {"groupName": "Series Name", "comicIds": ["id1", "id2"], "reason": "brief reason"}
]

If no groupable series found, return: []`, langName)

	// 构建候选列表（限制数量避免超出 token）
	maxCandidates := 100
	if len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}
	var lines []string
	for _, c := range candidates {
		lines = append(lines, fmt.Sprintf("- id:%s | title:%s", c.ID, c.Title))
	}

	userPrompt := fmt.Sprintf("Analyze these comic titles and group by series. Return a JSON array:\n\n%s", strings.Join(lines, "\n"))

	// 根据候选数量动态计算 MaxTokens（每本漫画约需 40-80 token 输出）
	estimatedTokens := len(candidates) * 80
	if estimatedTokens < 2048 {
		estimatedTokens = 2048
	}
	if estimatedTokens > 8192 {
		estimatedTokens = 8192
	}
	content, err := CallCloudLLM(cfg, systemPrompt, userPrompt, &LLMCallOptions{
		Scenario:  "ai_group_detect",
		MaxTokens: estimatedTokens,
	})
	if err != nil {
		return nil, err
	}

	// 清理并解析 JSON
	content = strings.ReplaceAll(content, "```json", "")
	content = strings.ReplaceAll(content, "```", "")
	content = strings.TrimSpace(content)

	// 防御空响应
	if content == "" {
		return nil, fmt.Errorf("AI returned empty response")
	}

	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start >= 0 && end > start {
		content = content[start : end+1]
	} else if start >= 0 {
		// 找到了 [ 但没有 ]，说明 JSON 被截断了，尝试修复
		content = repairTruncatedJSONArray(content[start:])
	} else {
		// 没有找到 JSON 数组
		return nil, fmt.Errorf("AI response does not contain a valid JSON array: %s", truncateStr(content, 200))
	}

	var suggestions []AIGroupSuggestion
	if err := json.Unmarshal([]byte(content), &suggestions); err != nil {
		// 二次尝试：修复后再解析
		repaired := repairTruncatedJSONArray(content)
		if err2 := json.Unmarshal([]byte(repaired), &suggestions); err2 != nil {
			log.Printf("[AI] Failed to parse group suggestions, raw: %s", truncateStr(content, 500))
			return nil, fmt.Errorf("failed to parse AI group suggestions: %w", err)
		}
	}

	// 过滤掉少于 2 本的分组
	var valid []AIGroupSuggestion
	for _, s := range suggestions {
		if len(s.ComicIDs) >= 2 {
			valid = append(valid, s)
		}
	}

	return valid, nil
}

// truncateStr 截断字符串到指定长度
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// repairTruncatedJSONArray 尝试修复被截断的 JSON 数组。
// 策略：找到最后一个完整的 JSON 对象（以 } 结尾），截断后面的不完整部分，补上 ]。
func repairTruncatedJSONArray(s string) string {
	s = strings.TrimSpace(s)
	// 如果已经是完整的 JSON 数组，直接返回
	if strings.HasSuffix(s, "]") {
		return s
	}

	// 找到最后一个 "}," 或 "}" 的位置
	lastBrace := strings.LastIndex(s, "}")
	if lastBrace < 0 {
		return "[]" // 完全无法修复
	}

	// 截取到最后一个完整对象
	result := strings.TrimSpace(s[:lastBrace+1])
	// 去掉尾部逗号
	result = strings.TrimRight(result, ", \t\n\r")
	// 补上 ]
	if !strings.HasSuffix(result, "]") {
		result += "]"
	}
	return result
}
