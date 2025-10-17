package rhythmgames

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

const cdnBase = "https://assets.koch2333.cn/backend/maimai/rating/"

var ratingLevels = [...]int{1000, 2000, 4000, 7000, 10000, 12000, 13000, 14000, 14500, 15000}

func RenderDXRatingSVG(rating int) string {
	if rating < 0 {
		rating = 0
	}
	r := clamp(rating, ratingLevels[0], ratingLevels[len(ratingLevels)-1])
	stage := 0
	for stage+1 < len(ratingLevels) && r >= ratingLevels[stage+1] {
		stage++
	}
	baseHref := fmt.Sprintf("%sUI_CMA_Rating_Base_%d.png", cdnBase, stage+1)
	digits := splitDigitsWithPad(rating)

	const (
		canvasW = 269.0
		canvasH = 70.0
		scale   = 0.8

		startX = 115.0
		stepX  = 28.0
		numW   = 34.0 * scale
		numH   = 40.0 * scale
		numY   = 20.0
	)

	var sb strings.Builder
	sb.Grow(4096)
	fmt.Fprintf(&sb, `<?xml version="1.0" encoding="UTF-8"?>
<svg width="%[1]g" height="%[2]g" viewBox="0 0 %[1]g %[2]g" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="DX Rating %[3]s">
  <defs></defs>
  <image href="%[4]s" x="0" y="0" width="%[1]g" height="%[2]g" image-rendering="auto"/>
`, canvasW, canvasH, html.EscapeString(strconv.Itoa(rating)), html.EscapeString(baseHref))

	for i, id := range digits {
		px := startX + float64(i)*stepX
		numHref := fmt.Sprintf("%sUI_CMN_Num_26p_%s.png", cdnBase, id)
		fmt.Fprintf(&sb, `  <image href="%s" x="%.2f" y="%.2f" width="%.2f" height="%.2f" image-rendering="auto"/>`+"\n",
			html.EscapeString(numHref), px, numY, numW, numH)
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func splitDigitsWithPad(v int) []string {
	s := strconv.Itoa(v)
	arr := make([]string, 0, 5)
	for _, ch := range s {
		arr = append(arr, string(ch))
	}
	for len(arr) < 5 {
		arr = append([]string{"10"}, arr...)
	}
	return arr
}
