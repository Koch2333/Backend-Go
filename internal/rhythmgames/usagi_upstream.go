package rhythmgames

import (
	"embed" // 开启 go:embed 能力（无需使用符号）
	"regexp"
	"strconv"
	"strings"
	_ "unsafe" // 有的 IDE 会误报 embed 未使用，保留这一行不影响
)

// 目录结构：
// internal/
//   rhythmgames/
//     usagi_upstream.go   ← 本文件
//     assets/
//       DXRating.vue      ← 上游源码（原封不动）

//go:embed assets/DXRating.vue
var upstreamDXRatingVue string

// 使用仓库内嵌的上游 DXRating.vue，取出 <svg>…</svg>，删掉 Vue 指令属性，
// 把与 “rating” 相关的插值替换成实际分数。成功返回完整 SVG，失败返回空串。
func RenderUsagiDXBadgeExactUpstream(rating int) string {
	s := strings.TrimSpace(upstreamDXRatingVue)
	if s == "" {
		return ""
	}

	// 1) 提取 <template>…</template>；没有就取第一个 <svg>…</svg>
	tpl := extractBetweenInsensitive(s, "<template>", "</template>")
	if tpl == "" {
		reSVG := regexp.MustCompile(`(?is)<svg[\s\S]*?</svg>`)
		if m := reSVG.FindString(s); m != "" {
			tpl = m
		} else {
			return ""
		}
	}

	// 2) 仅保留第一段 <svg>…</svg>
	reSVG := regexp.MustCompile(`(?is)<svg[\s\S]*?</svg>`)
	svg := reSVG.FindString(tpl)
	if svg == "" {
		return ""
	}

	// 3) 去掉常见 Vue 指令属性（v- / : / @）
	reAttr := regexp.MustCompile(`\s(?:v-[\w:-]+|:[\w:-]+|@[\w:-]+)="[^"]*"`)
	svg = reAttr.ReplaceAllString(svg, "")

	// 4) 替换包含 "rating" 的插值为实际分数
	if rating < 0 {
		rating = 0
	}
	reMustache := regexp.MustCompile(`\{\{[^}]*rating[^}]*\}\}`)
	svg = reMustache.ReplaceAllString(svg, strconv.Itoa(rating))

	// 5) 兜底：若模板是纯文本数字，替换第一处 3~6 位连续数字（避免误伤颜色/坐标）
	reDigits := regexp.MustCompile(`(\D)(\d{3,6})(\D)`)
	svg = reDigits.ReplaceAllStringFunc(svg, func(t string) string {
		sub := reDigits.FindStringSubmatch(t)
		if len(sub) == 4 {
			return sub[1] + strconv.Itoa(rating) + sub[3]
		}
		return t
	})

	// 6) 简单规范空白
	svg = strings.ReplaceAll(svg, "\r\n", "\n")
	reSpace := regexp.MustCompile(`\s{2,}`)
	svg = reSpace.ReplaceAllString(svg, " ")

	if !strings.Contains(svg, "<svg") {
		return ""
	}
	return strings.TrimSpace(svg)
}

// 工具：大小写不敏感提取 [start, end] 之间的子串（不含边界）
func extractBetweenInsensitive(s, start, end string) string {
	ls := strings.ToLower(s)
	lstart := strings.ToLower(start)
	lend := strings.ToLower(end)
	i := strings.Index(ls, lstart)
	if i < 0 {
		return ""
	}
	j := strings.Index(ls[i+len(lstart):], lend)
	if j < 0 {
		return ""
	}
	return s[i+len(start) : i+len(start)+j]
}
