package rhythmgames

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Theme string

const (
	ThemeDark  Theme = "dark"
	ThemeLight Theme = "light"
)

type Palette struct {
	BG, FG, Sub, GradFrom, GradTo string
}

var dark = &Palette{
	BG: "#0b0b0e", FG: "#fafafa", Sub: "#9aa0a6",
	GradFrom: "#7c4dff", GradTo: "#00e5ff",
}
var light = &Palette{
	BG: "#ffffff", FG: "#111827", Sub: "#6b7280",
	GradFrom: "#6366f1", GradTo: "#22d3ee",
}

type GameKey string

type UserID struct {
	Username string
}

type RatingResult struct {
	DisplayName string
	Rating      int
	Meta        map[string]any
}

type Provider interface {
	Key() GameKey
	FetchRating(ctx context.Context, id UserID) (*RatingResult, error)
	ThemePalette() *Palette
}

var (
	ErrUserNotFound = fmt.Errorf("user not found")
	providers       = map[GameKey]Provider{}
)

func Register(p Provider) {
	k := p.Key()
	if k == "" {
		panic("rhythmgames: empty provider key")
	}
	if _, ok := providers[k]; ok {
		panic("rhythmgames: duplicate provider " + string(k))
	}
	providers[k] = p
}

func GetProvider(game string) (Provider, error) {
	if p, ok := providers[GameKey(strings.ToLower(strings.TrimSpace(game)))]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("unknown game: %s", game)
}

/* ---------- UsagiPass 风格徽章渲染（纯 SVG，无位图） ---------- */

// RenderUsagiDXBadge 生成 UsagiPass 顶部“DX Rating”风格的黄色分数框。
// 组成：外渐变描边胶囊 + 内黄牌（纵向渐变、轻噪声、内阴影、顶部高光）+ 左侧“DX”蓝圆徽 + “DX Rating”文本 + 数字分格。
func RenderUsagiDXBadge(rating int) string {
	if rating < 0 {
		rating = 0
	}
	txt := strconv.Itoa(rating)

	// 尺寸估算：左侧徽+标签≈110，数字每位 24，整体左右留白≈18
	leftBlock := 110
	boxW := 24
	w := leftBlock + len(txt)*boxW + 18
	h := 46

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg width="%d" height="%d" viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="DX Rating %s">
  <defs>
    <!-- 外描边蓝青渐变 -->
    <linearGradient id="rg_dx_outer" x1="0" y1="0" x2="1" y2="0">
      <stop offset="0%%" stop-color="#5b7cfa"/>
      <stop offset="100%%" stop-color="#22d3ee"/>
    </linearGradient>

    <!-- 黄牌纵向渐变（上亮下稍深） -->
    <linearGradient id="rg_dx_yellow" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%%"  stop-color="#ffe168"/>
      <stop offset="55%%" stop-color="#f7d21a"/>
      <stop offset="100%%" stop-color="#f0c80e"/>
    </linearGradient>

    <!-- 轻噪声纹理（极淡） -->
    <filter id="rg_dx_noise" x="-20%%" y="-20%%" width="140%%" height="140%%">
      <feTurbulence type="fractalNoise" baseFrequency="0.8" numOctaves="2" stitchTiles="stitch" result="n"/>
      <feColorMatrix in="n" type="matrix" values="
        0 0 0 0 0
        0 0 0 0 0
        0 0 0 0 0
        0 0 0 .04 0" result="noiseA"/>
      <feBlend in="SourceGraphic" in2="noiseA" mode="multiply"/>
    </filter>

    <!-- 内阴影 -->
    <filter id="rg_dx_inner" x="-20%%" y="-20%%" width="140%%" height="140%%">
      <feOffset dx="0" dy="1" />
      <feGaussianBlur stdDeviation="0.8" result="o"/>
      <feComposite in="o" in2="SourceAlpha" operator="arithmetic" k2="-1" k3="1"/>
      <feColorMatrix type="matrix" values="
        0 0 0 0 0
        0 0 0 0 0
        0 0 0 0 0
        0 0 0 .35 0"/>
      <feBlend in="SourceGraphic" mode="normal"/>
    </filter>

    <!-- 数字轻阴影 -->
    <filter id="rg_dx_digit" x="-50%%" y="-50%%" width="200%%" height="200%%">
      <feDropShadow dx="0" dy="1" stdDeviation="1" flood-color="#000" flood-opacity="0.25"/>
    </filter>

    <!-- “DX” 徽章的蓝色渐变 -->
    <linearGradient id="rg_dx_blue" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%%" stop-color="#45a2ff"/>
      <stop offset="100%%" stop-color="#2a72ff"/>
    </linearGradient>

    <style>
      .lbl { font: 700 13px ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto; fill: #2b2b2b; letter-spacing: .4px; }
      .dx  { font: 800 12px ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto; fill:#fff; }
      .num { font: 800 22px ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto; fill: #1a1a1a; }
    </style>
  </defs>

  <!-- 外层渐变描边胶囊 -->
  <rect x="1.5" y="1.5" rx="18" ry="18" width="%d" height="%d" fill="none" stroke="url(#rg_dx_outer)" stroke-width="3"/>

  <!-- 内部黄牌：渐变 + 轻噪声 + 内描边 + 高光 + 内阴影 -->
  <g filter="url(#rg_dx_inner)">
    <rect x="5" y="5" rx="14" ry="14" width="%d" height="%d" fill="url(#rg_dx_yellow)"/>
    <rect x="5" y="5" rx="14" ry="14" width="%d" height="%d" fill="transparent" filter="url(#rg_dx_noise)"/>
    <rect x="5.5" y="5.5" rx="13.5" ry="13.5" width="%d" height="%d" fill="none" stroke="#f2c94c" stroke-width="1"/>
    <!-- 顶部高光（弧形带透明） -->
    <path d="M 8 14 Q %d 0 %d 14 L %d 18 Q %d 6 %d 18 Z" fill="rgba(255,255,255,0.28)"/>
  </g>

  <!-- 左侧蓝色 “DX” 圆徽 -->
  <g transform="translate(18, 11)">
    <circle cx="12" cy="12" r="12" fill="url(#rg_dx_blue)" stroke="#1c4ddb" stroke-width="1"/>
    <text class="dx" x="12" y="15" text-anchor="middle">DX</text>
  </g>

  <!-- “DX Rating” 标签文字 -->
  <g transform="translate(44, 28)">
    <text class="lbl" text-anchor="start">DX Rating</text>
  </g>

  <!-- 数字分格（从 leftBlock=110 起排） -->
  %s
</svg>`,
		w, h, w, h, html.EscapeString(txt),
		// 外描边
		w-3, h-3,
		// 主牌
		w-10, h-10,
		// 噪声层
		w-10, h-10,
		// 内描边
		w-11, h-11,
		// 高光控制点（弧形顶端）
		w/2, w-8, w-8, w/2, 8,
		renderDigits(txt, leftBlock, 10), // y=10 更贴近原版位置
	)
}

// renderDigits：逐位绘制数字格（浅阴影+圆角格+轻描边），视觉靠近 UsagiPass。
func renderDigits(s string, startX, y int) string {
	var b strings.Builder
	x := startX
	boxW := 24
	boxH := 26
	for i, ch := range s {
		// 适度增加数字间距（与 UsagiPass 接近）
		if i > 0 {
			x += 1
		}
		fmt.Fprintf(&b, `
  <g transform="translate(%d, %d)">
    <rect x="0" y="0" width="%d" height="%d" rx="6" ry="6"
          fill="#ffe680" stroke="#f2c94c" stroke-width="1"/>
    <text class="num" x="%d" y="20" text-anchor="middle" filter="url(#rg_dx_digit)">%c</text>
  </g>`, x, y, boxW, boxH, boxW/2, ch)
		x += boxW
	}
	return b.String()
}

/* ----------（可留存：通用卡片渲染，其他场景需要再用） ---------- */

func RenderCardSVG(title, displayName string, rating int, theme Theme, game *Palette) string {
	pa := mergePalette(pickBase(theme), game)
	nn := html.EscapeString(displayName)
	rat := strconv.Itoa(rating)

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg width="520" height="120" viewBox="0 0 520 120" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="%s: %s">
  <defs>
    <linearGradient id="g" x1="0" y1="0" x2="1" y2="0">
      <stop offset="0%%" stop-color="%s"/><stop offset="100%%" stop-color="%s"/>
    </linearGradient>
    <style>
      .title{font:700 16px ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto;fill:%s}
      .name{font:600 14px ui-sans-serif,system-ui;fill:%s}
      .num{font:800 42px ui-sans-serif,system-ui;fill:%s}
      .sub{font:500 12px ui-sans-serif,system-ui;fill:%s}
      .card{fill:%s}
    </style>
  </defs>
  <rect class="card" x="0" y="0" width="520" height="120" rx="14"/>
  <text class="title" x="20" y="34">%s</text>
  <text class="name"  x="20" y="56">@%s</text>
  <text class="num"   x="500" y="66" text-anchor="end">%s</text>
  <text class="sub"   x="500" y="86" text-anchor="end">by RhythmGames</text>
  <rect x="20" y="94" width="480" height="12" rx="6" fill="url(#g)"/>
</svg>`,
		title, rat,
		pa.GradFrom, pa.GradTo,
		pa.FG, pa.Sub, pa.FG, pa.Sub, pa.BG,
		title, nn, rat)
}

func pickBase(theme Theme) *Palette {
	switch strings.ToLower(string(theme)) {
	case "light":
		return light
	default:
		return dark
	}
}

func mergePalette(base, game *Palette) *Palette {
	if game == nil {
		return base
	}
	out := *base
	if game.BG != "" {
		out.BG = game.BG
	}
	if game.FG != "" {
		out.FG = game.FG
	}
	if game.Sub != "" {
		out.Sub = game.Sub
	}
	if game.GradFrom != "" {
		out.GradFrom = game.GradFrom
	}
	if game.GradTo != "" {
		out.GradTo = game.GradTo
	}
	return &out
}

var UpstreamDXSource = "https://raw.githubusercontent.com/TrueRou/UsagiPass/main/web/src/components/DXRating.vue"

// http 客户端（带超时）
var upstreamHTTP = &http.Client{Timeout: 8 * time.Second}

// RenderUsagiDXBadgeExact 从上游 DXRating.vue 抓取 <template> 内的 SVG 片段，
// 尽可能“按原样”输出，只把分数字符串替换为 rating。
// 如果抓取或解析失败，返回空字符串让调用方回退到本地渲染。
func RenderUsagiDXBadgeExact(ctx context.Context, rating int) string {
	src := UpstreamDXSource
	req, err := http.NewRequestWithContext(ctx, "GET", src, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "RhythmGames-DXRating/1.0 (+github)")

	resp, err := upstreamHTTP.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	raw := string(b)

	// 1) 尝试提取 <template> ... </template>
	tpl := extractBetween(raw, "<template>", "</template>")
	if tpl == "" {
		// 兼容某些风格：直接找 <svg>...</svg>
		tpl = extractBetween(raw, "<svg", "</svg>")
		if tpl != "" {
			tpl = "<svg" + tpl + "</svg>"
		}
	}
	if tpl == "" {
		return ""
	}

	// 2) 处理 Vue 指令与插值，使其成为静态 SVG
	out := vueTemplateToStaticSVG(tpl, rating)

	// 3) 粗检：必须包含 <svg ...> 才算成功
	if !strings.Contains(out, "<svg") || !strings.Contains(out, "</svg>") {
		return ""
	}
	return out
}

// —— 工具：从 s 中提取 firstStart 到 firstEnd（包含 end）的子串（不区分大小写）
func extractBetween(s, start, end string) string {
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

// —— 把 Vue 的 template 转为静态 SVG：极小心清理常见指令与插值
func vueTemplateToStaticSVG(tpl string, rating int) string {
	// 去掉最外层的换行/缩进
	s := strings.TrimSpace(tpl)

	// 统一换行，防止正则跨行失效
	s = strings.ReplaceAll(s, "\r\n", "\n")

	// 1) 删除常见 Vue 指令属性（v-、:、@）
	//   例：v-if="..." v-show="..." :class="..." @click="..."
	reAttr := regexp.MustCompile(`\s(?:v-[\w:-]+|:[\w:-]+|@[\w:-]+)="[^"]*"`)
	s = reAttr.ReplaceAllString(s, "")

	// 2) 删除自定义组件标签（保守做法：保留原生 svg 标签），
	//    这里只在模板里保留 <svg>...，其他多余容器（如 <div>）尽量剥离
	//    直接抽出第一个 <svg ...>...</svg>
	reSVG := regexp.MustCompile(`(?is)<svg[\s\S]*?</svg>`)
	m := reSVG.FindString(s)
	if m != "" {
		s = m
	}

	// 3) 把所有包含“rating”字样的 Mustache 插值替换为数字
	//    例：{{ rating }} / {{ props.rating }} / {{ user.rating }}
	reMustache := regexp.MustCompile(`\{\{[^}]*rating[^}]*\}\}`)
	s = reMustache.ReplaceAllString(s, strconv.Itoa(rating))

	// 4) 清理残留的 Mustache（不含 rating 的），避免原样输出花括号
	reAnyMustache := regexp.MustCompile(`\{\{[^}]+\}\}`)
	s = reAnyMustache.ReplaceAllString(s, "")

	// 5) 兜底：若模板并非采用插值而是文本节点“DX Rating 12345”，尝试替换连续数字
	//    只替换第一处 3~6 位连续数字（避免误伤颜色/坐标等）
	reDigits := regexp.MustCompile(`(\D)(\d{3,6})(\D)`)
	s = reDigits.ReplaceAllStringFunc(s, func(t string) string {
		sub := reDigits.FindStringSubmatch(t)
		if len(sub) == 4 {
			return sub[1] + strconv.Itoa(rating) + sub[3]
		}
		return t
	})

	// 6) 防御性修剪重复空白
	s = strings.ReplaceAll(s, "\t", " ")
	reSpace := regexp.MustCompile(`\s{2,}`)
	s = reSpace.ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
}
