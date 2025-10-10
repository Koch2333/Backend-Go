package rhythmgames

import (
	"embed"
	"encoding/base64"
	"fmt"
	"html"
	"strconv"
	"strings"
)

// ------------------------------------------------------------------
// 用通配符一次性内嵌，避免某一行写错路径导致 len=0
// ------------------------------------------------------------------

//go:embed assets/rating/UI_CMA_Rating_Base_*.png
var baseFS embed.FS

//go:embed assets/rating/UI_CMN_Num_26p_*.png
var numFS embed.FS

// 与前端一致的分段
var ratingLevels = [...]int{1000, 2000, 4000, 7000, 10000, 12000, 13000, 14000, 14500, 15000}

// ======================= 对外渲染入口 =======================

// RenderDXRatingSVG 生成 “一模一样”的 DX Rating 框（SVG 内嵌 PNG）
func RenderDXRatingSVG(rating int) string {
	if rating < 0 {
		rating = 0
	}

	// 1) 选底图（stage 1..10）
	r := clamp(rating, ratingLevels[0], ratingLevels[len(ratingLevels)-1])
	stage := 0
	for stage+1 < len(ratingLevels) && r >= ratingLevels[stage+1] {
		stage++
	}
	basePNG := loadBasePNG(stage + 1) // <-- 改这里

	// 2) 5 位字符串（不足补 '10'）
	digits := splitDigitsWithPad(rating)

	// 3) 画布尺寸/坐标与前端一致
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

	// 4) 组装 SVG
	var sb strings.Builder
	sb.Grow(4096)
	fmt.Fprintf(&sb, `<?xml version="1.0" encoding="UTF-8"?>
<svg width="%[1]g" height="%[2]g" viewBox="0 0 %[1]g %[2]g" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="DX Rating %[3]s">
  <defs></defs>
  <!-- 背景底图 -->
  <image href="data:image/png;base64,%[4]s" x="0" y="0" width="%[1]g" height="%[2]g" image-rendering="auto"/>
`, canvasW, canvasH, html.EscapeString(strconv.Itoa(rating)), b64(basePNG))

	// 5) 逐位数字
	for i, id := range digits {
		px := startX + float64(i)*stepX
		png := loadNumPNG(id) // <-- 改这里
		if len(png) == 0 {
			// 可选：打印一次帮助定位缺图
			fmt.Printf("[dxrating] missing digit asset: %s\n", id)
			continue
		}
		fmt.Fprintf(&sb, `  <image href="data:image/png;base64,%s" x="%.2f" y="%.2f" width="%.2f" height="%.2f" image-rendering="auto"/>`+"\n",
			b64(png), px, numY, numW, numH)
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}

// ======================= 资源读取 =======================

// 资源读取（容错版）——放进你现有的 usagi_dx_exact.go 覆盖原实现

func loadBasePNG(stage int) []byte {
	// 1) 先按常规路径读
	name := fmt.Sprintf("assets/rating/UI_CMA_Rating_Base_%d.png", stage)
	if b, err := baseFS.ReadFile(name); err == nil && len(b) > 0 {
		return b
	}
	// 2) 若上游素材被放到 rating/base/ 或大小写细节不同，则目录内模糊搜
	if b := fuzzyRead(baseFS, "assets/rating", fmt.Sprintf("UI_CMA_Rating_Base_%d", stage)); len(b) > 0 {
		return b
	}
	if b := fuzzyRead(baseFS, "assets/rating/base", fmt.Sprintf("UI_CMA_Rating_Base_%d", stage)); len(b) > 0 {
		return b
	}
	fmt.Println("[dxrating] base not found:", name)
	return nil
}

func loadNumPNG(id string) []byte {
	// 1) 常规路径（推荐用法）
	name := fmt.Sprintf("assets/rating/UI_CMN_Num_26p_%s.png", id)
	if b, err := numFS.ReadFile(name); err == nil && len(b) > 0 {
		return b
	}
	// 2) 兼容 assets/rating/num/ 放法
	alt := fmt.Sprintf("assets/rating/num/UI_CMN_Num_26p_%s.png", id)
	if b, err := numFS.ReadFile(alt); err == nil && len(b) > 0 {
		return b
	}
	// 3) 再模糊搜（防止命名差异、大小写差异）
	if b := fuzzyRead(numFS, "assets/rating", "UI_CMN_Num_26p_"+id); len(b) > 0 {
		return b
	}
	if b := fuzzyRead(numFS, "assets/rating/num", "UI_CMN_Num_26p_"+id); len(b) > 0 {
		return b
	}
	fmt.Println("[dxrating] digit not found:", name, "(or", alt, ")")
	return nil
}

// 在同文件里补这个工具函数
func fuzzyRead(fs embed.FS, dir, mustContain string) []byte {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.Contains(name, mustContain) && strings.HasSuffix(strings.ToLower(name), ".png") {
			if b, err := fs.ReadFile(dir + "/" + name); err == nil && len(b) > 0 {
				return b
			}
		}
	}
	return nil
}

// ======================= 小工具 =======================

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

func b64(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}
