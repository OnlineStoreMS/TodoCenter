package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"todocenter/internal/feishu"
	"todocenter/internal/model"
	"todocenter/internal/repo"
)

// NotifyLevelDTO 重复提醒等级：warning → critical → imminent。
// beforeMinutes：距截止 ≤ 该分钟数时进入对应等级。
type NotifyLevelDTO struct {
	Key           string `json:"key"`
	Label         string `json:"label"`
	Enabled       bool   `json:"enabled"`
	BeforeMinutes int    `json:"beforeMinutes"`
}

type NotifyConfigDTO struct {
	Enabled             bool             `json:"enabled"`
	WebhookURL          string           `json:"webhookUrl"`
	SecretSet           bool             `json:"secretSet"`
	PollIntervalMinutes int              `json:"pollIntervalMinutes"`
	Levels              []NotifyLevelDTO `json:"levels"`
}

type NotifyStateDTO struct {
	LastRunAt     string `json:"lastRunAt,omitempty"`
	LastRunOK     bool   `json:"lastRunOk"`
	LastError     string `json:"lastError,omitempty"`
	LastSentCount int    `json:"lastSentCount"`
}

type NotifySaveReq struct {
	Enabled             *bool             `json:"enabled"`
	WebhookURL          *string           `json:"webhookUrl"`
	Secret              *string           `json:"secret"`
	PollIntervalMinutes *int              `json:"pollIntervalMinutes"`
	Levels              *[]NotifyLevelDTO `json:"levels"`
}

type NotifyService struct {
	repos  *repo.Repos
	todos  *TodoService
	feishu *feishu.Client
}

func NewNotifyService(repos *repo.Repos, todos *TodoService) *NotifyService {
	return &NotifyService{
		repos:  repos,
		todos:  todos,
		feishu: feishu.NewClient(),
	}
}

func (s *NotifyService) Get(tenantID uint64) (*NotifyConfigDTO, *NotifyStateDTO, error) {
	row, err := s.repos.Notify.GetOrCreate(tenantID)
	if err != nil {
		return nil, nil, err
	}
	return toNotifyConfig(row), toNotifyState(row), nil
}

func (s *NotifyService) Save(tenantID uint64, req NotifySaveReq) (*NotifyConfigDTO, *NotifyStateDTO, error) {
	row, err := s.repos.Notify.GetOrCreate(tenantID)
	if err != nil {
		return nil, nil, err
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.WebhookURL != nil {
		row.WebhookURL = strings.TrimSpace(*req.WebhookURL)
	}
	if req.Secret != nil {
		sec := strings.TrimSpace(*req.Secret)
		if sec != "" {
			row.Secret = sec
		}
	}
	if req.PollIntervalMinutes != nil {
		row.PollIntervalMinutes = normalizePollInterval(*req.PollIntervalMinutes)
	}
	if req.Levels != nil {
		levels := normalizeLevels(*req.Levels)
		if err := validateLevelOrder(levels); err != nil {
			return nil, nil, fmt.Errorf("%w: %s", ErrBadRequest, err.Error())
		}
		raw, _ := json.Marshal(levels)
		row.LevelsJSON = string(raw)
		row.LeadMinutes = maxPreDueMinutes(levels)
		row.NotifyOverdue = false
	}
	if err := s.repos.Notify.Save(row); err != nil {
		return nil, nil, err
	}
	return toNotifyConfig(row), toNotifyState(row), nil
}

func (s *NotifyService) Test(ctx context.Context, tenantID uint64, text string) error {
	row, err := s.repos.Notify.GetOrCreate(tenantID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(row.WebhookURL) == "" {
		return fmt.Errorf("%w: 请先填写并保存飞书 Webhook", ErrBadRequest)
	}
	if strings.TrimSpace(text) == "" {
		text = "【待办中心】飞书通知测试消息"
	}
	return s.feishu.SendText(ctx, row.WebhookURL, row.Secret, text)
}

func (s *NotifyService) ResetState(tenantID uint64) (*NotifyConfigDTO, *NotifyStateDTO, error) {
	row, err := s.repos.Notify.GetOrCreate(tenantID)
	if err != nil {
		return nil, nil, err
	}
	row.NotifiedJSON = ""
	row.LastError = ""
	row.LastSentCount = 0
	row.LastRunOK = false
	row.LastRunAt = nil
	if err := s.repos.Notify.Save(row); err != nil {
		return nil, nil, err
	}
	return toNotifyConfig(row), toNotifyState(row), nil
}

func (s *NotifyService) RunOnce(ctx context.Context, tenantID uint64) (int, error) {
	row, err := s.repos.Notify.GetOrCreate(tenantID)
	if err != nil {
		return 0, err
	}
	sent, runErr := s.runTenant(ctx, row)
	errMsg := ""
	ok := runErr == nil
	if runErr != nil {
		errMsg = runErr.Error()
	}
	_ = s.repos.Notify.UpdateRunState(tenantID, ok, errMsg, sent)
	return sent, runErr
}

func (s *NotifyService) RunAllEnabled(ctx context.Context) {
	list, err := s.repos.Notify.ListEnabled()
	if err != nil {
		return
	}
	for i := range list {
		row := &list[i]
		sent, runErr := s.runTenant(ctx, row)
		errMsg := ""
		ok := runErr == nil
		if runErr != nil {
			errMsg = runErr.Error()
		}
		_ = s.repos.Notify.UpdateRunState(row.TenantID, ok, errMsg, sent)
	}
}

func (s *NotifyService) MinPollInterval() time.Duration {
	list, err := s.repos.Notify.ListEnabled()
	if err != nil || len(list) == 0 {
		return 5 * time.Minute
	}
	minM := 0
	for _, row := range list {
		m := normalizePollInterval(row.PollIntervalMinutes)
		if minM == 0 || m < minM {
			minM = m
		}
	}
	if minM < 1 {
		minM = 1
	}
	return time.Duration(minM) * time.Minute
}

func (s *NotifyService) runTenant(ctx context.Context, row *model.TenantTodoNotify) (int, error) {
	if row == nil || !row.Enabled || strings.TrimSpace(row.WebhookURL) == "" {
		return 0, nil
	}
	_ = s.todos.EnsureMonthlyInstances(row.TenantID)

	levels := resolveLevels(row)
	maxLead := maxPreDueMinutes(levels)
	now := time.Now()
	until := now.Add(time.Duration(maxLead) * time.Minute)
	todos, err := s.repos.Todo.ListDueForNotify(row.TenantID, until)
	if err != nil {
		return 0, err
	}

	notified := parseNotifiedMap(row.NotifiedJSON)
	cats, _ := s.repos.Category.ListAll(row.TenantID)
	catName := map[uint64]string{}
	for _, c := range cats {
		catName[c.ID] = c.Name
	}

	todayKey := now.Format("2006-01-02")
	sent := 0
	var sendErr error

	for i := range todos {
		t := &todos[i]
		if t.DueAt == nil {
			continue
		}
		due := t.DueAt.In(now.Location())
		levelKey := currentNotifyLevel(now, due, levels)
		if levelKey == "" {
			continue
		}
		lvl := findLevel(levels, levelKey)
		if lvl == nil || !lvl.Enabled {
			continue
		}

		dueStamp := due.Format("200601021504")
		dedupKey := fmt.Sprintf("%d:%s:%s", t.ID, levelKey, dueStamp)
		if _, ok := notified[dedupKey]; ok {
			continue
		}

		card := buildTodoNotificationCard(t, catName[t.CategoryID], levelKey, now, due)
		if err := s.feishu.SendInteractiveCard(ctx, row.WebhookURL, row.Secret, card); err != nil {
			sendErr = err
			break
		}
		notified[dedupKey] = todayKey
		sent++
	}

	row.NotifiedJSON = marshalNotified(pruneNotified(notified, 60))
	_ = s.repos.Notify.Save(row)
	return sent, sendErr
}

// buildTodoNotificationCard 样式对齐 StoreSyncAgent 售后通知：标题「待办通知 · 场景」+ **标签：** 值。
func buildTodoNotificationCard(t *model.Todo, categoryName, levelKey string, now, due time.Time) feishu.InteractiveCard {
	kind := "普通待办"
	if t.ParentID > 0 {
		kind = "月待办"
		if t.PeriodKey != "" {
			kind = "月待办 · " + t.PeriodKey
		}
	}

	var lines []string
	if line := mdLine("标题", t.Title, ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("分类", categoryName, "blue"); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("类型", kind, "purple"); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("状态", statusLabel(t.Status), ""); line != "" {
		lines = append(lines, line)
	}
	prioColor := ""
	if t.Priority == model.TodoPriorityHigh {
		prioColor = "red"
	} else if t.Priority == model.TodoPriorityLow {
		prioColor = "grey"
	}
	if line := mdLine("优先级", priorityLabel(t.Priority), prioColor); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("截止时间", due.Format("2006-01-02 15:04"), ""); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("提醒等级", levelShortLabel(levelKey), levelValueColor(levelKey)); line != "" {
		lines = append(lines, line)
	}
	if line := mdLine("时效", remainingText(now, due), levelValueColor(levelKey)); line != "" {
		lines = append(lines, line)
	}
	if desc := strings.TrimSpace(t.Description); desc != "" {
		if line := mdLine("说明", truncateRunes(desc, 120), "grey"); line != "" {
			lines = append(lines, line)
		}
	}

	return feishu.InteractiveCard{
		Title:    "待办通知 · " + levelShortLabel(levelKey),
		Template: levelCardTemplate(levelKey),
		Markdown: strings.Join(lines, "\n"),
	}
}

func mdLine(label, value, color string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = escapeLarkMD(value)
	if color != "" {
		value = fmt.Sprintf("<font color='%s'>%s</font>", color, value)
	}
	return fmt.Sprintf("**%s：** %s", escapeLarkMD(label), value)
}

func escapeLarkMD(s string) string {
	return strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(s)
}

func remainingText(now, due time.Time) string {
	d := due.Sub(now)
	if d <= 0 {
		return "已到期"
	}
	h := int(d / time.Hour)
	m := int((d % time.Hour) / time.Minute)
	if h > 0 && m > 0 {
		return fmt.Sprintf("剩余 %dh%dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("剩余 %dh", h)
	}
	if m > 0 {
		return fmt.Sprintf("剩余 %dm", m)
	}
	return "剩余不足 1m"
}

func truncateRunes(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if max <= 0 || len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "…"
}

func statusLabel(s string) string {
	switch s {
	case model.TodoStatusPending:
		return "待处理"
	case model.TodoStatusInProgress:
		return "进行中"
	case model.TodoStatusDone:
		return "已完成"
	case model.TodoStatusCancelled:
		return "已取消"
	default:
		return s
	}
}

func levelShortLabel(key string) string {
	switch key {
	case model.NotifyLevelWarning:
		return "预警"
	case model.NotifyLevelCritical:
		return "紧急"
	case model.NotifyLevelImminent:
		return "临期"
	default:
		return key
	}
}

func levelValueColor(key string) string {
	switch key {
	case model.NotifyLevelImminent, model.NotifyLevelCritical:
		return "red"
	case model.NotifyLevelWarning:
		return "orange"
	default:
		return ""
	}
}

func levelCardTemplate(key string) string {
	switch key {
	case model.NotifyLevelImminent:
		return "red"
	case model.NotifyLevelCritical:
		return "orange"
	case model.NotifyLevelWarning:
		return "orange"
	default:
		return "blue"
	}
}

func currentNotifyLevel(now, due time.Time, levels []NotifyLevelDTO) string {
	remaining := due.Sub(now)
	if remaining <= 0 {
		return ""
	}
	remainMin := int(remaining / time.Minute)
	if remainMin < 0 {
		remainMin = 0
	}
	preOrder := []string{model.NotifyLevelImminent, model.NotifyLevelCritical, model.NotifyLevelWarning}
	for _, key := range preOrder {
		lvl := findLevel(levels, key)
		if lvl == nil || !lvl.Enabled || lvl.BeforeMinutes <= 0 {
			continue
		}
		if remainMin <= lvl.BeforeMinutes {
			return key
		}
	}
	return ""
}

func defaultLevels() []NotifyLevelDTO {
	return []NotifyLevelDTO{
		{Key: model.NotifyLevelWarning, Label: levelLabel(model.NotifyLevelWarning), Enabled: true, BeforeMinutes: 24 * 60},
		{Key: model.NotifyLevelCritical, Label: levelLabel(model.NotifyLevelCritical), Enabled: true, BeforeMinutes: 4 * 60},
		{Key: model.NotifyLevelImminent, Label: levelLabel(model.NotifyLevelImminent), Enabled: true, BeforeMinutes: 30},
	}
}

func resolveLevels(row *model.TenantTodoNotify) []NotifyLevelDTO {
	if row != nil && strings.TrimSpace(row.LevelsJSON) != "" {
		var levels []NotifyLevelDTO
		if err := json.Unmarshal([]byte(row.LevelsJSON), &levels); err == nil && len(levels) > 0 {
			return normalizeLevels(levels)
		}
	}
	if row != nil && row.LeadMinutes > 0 {
		lead := normalizeLeadMinutes(row.LeadMinutes)
		crit := lead
		if crit > 4*60 {
			crit = 4 * 60
		}
		imm := lead
		if imm > 30 {
			imm = 30
		}
		if imm > crit {
			imm = crit
		}
		warn := lead
		if warn < crit {
			warn = crit
		}
		return normalizeLevels([]NotifyLevelDTO{
			{Key: model.NotifyLevelWarning, Enabled: warn > crit, BeforeMinutes: warn},
			{Key: model.NotifyLevelCritical, Enabled: crit > imm, BeforeMinutes: crit},
			{Key: model.NotifyLevelImminent, Enabled: true, BeforeMinutes: imm},
		})
	}
	return defaultLevels()
}

func normalizeLevels(in []NotifyLevelDTO) []NotifyLevelDTO {
	byKey := map[string]NotifyLevelDTO{}
	for _, l := range in {
		key := strings.ToLower(strings.TrimSpace(l.Key))
		switch key {
		case model.NotifyLevelWarning, model.NotifyLevelCritical, model.NotifyLevelImminent:
			l.Key = key
			l.Label = levelLabel(key)
			l.BeforeMinutes = normalizeLeadMinutes(l.BeforeMinutes)
			byKey[key] = l
		}
	}
	out := defaultLevels()
	for i := range out {
		if v, ok := byKey[out[i].Key]; ok {
			out[i].Enabled = v.Enabled
			out[i].BeforeMinutes = v.BeforeMinutes
		}
	}
	return out
}

func validateLevelOrder(levels []NotifyLevelDTO) error {
	warn := findLevel(levels, model.NotifyLevelWarning)
	crit := findLevel(levels, model.NotifyLevelCritical)
	imm := findLevel(levels, model.NotifyLevelImminent)
	var enabled []NotifyLevelDTO
	for _, l := range []*NotifyLevelDTO{warn, crit, imm} {
		if l != nil && l.Enabled && l.BeforeMinutes > 0 {
			enabled = append(enabled, *l)
		}
	}
	sort.Slice(enabled, func(i, j int) bool { return enabled[i].BeforeMinutes < enabled[j].BeforeMinutes })
	// 启用的等级应按：imminent ≤ critical ≤ warning
	orderIdx := map[string]int{
		model.NotifyLevelImminent: 0,
		model.NotifyLevelCritical: 1,
		model.NotifyLevelWarning:  2,
	}
	for i := 0; i < len(enabled)-1; i++ {
		if orderIdx[enabled[i].Key] > orderIdx[enabled[i+1].Key] {
			return fmt.Errorf("等级时间需满足：imminent ≤ critical ≤ warning（距截止越近阈值越小）")
		}
		if enabled[i].BeforeMinutes > enabled[i+1].BeforeMinutes {
			return fmt.Errorf("等级时间需满足：imminent ≤ critical ≤ warning（距截止越近阈值越小）")
		}
	}
	return nil
}

func findLevel(levels []NotifyLevelDTO, key string) *NotifyLevelDTO {
	for i := range levels {
		if levels[i].Key == key {
			return &levels[i]
		}
	}
	return nil
}

func maxPreDueMinutes(levels []NotifyLevelDTO) int {
	max := 0
	for _, l := range levels {
		if !l.Enabled {
			continue
		}
		if l.BeforeMinutes > max {
			max = l.BeforeMinutes
		}
	}
	return max
}

func levelLabel(key string) string {
	switch key {
	case model.NotifyLevelWarning:
		return "Warning 预警"
	case model.NotifyLevelCritical:
		return "Critical 紧急"
	case model.NotifyLevelImminent:
		return "Imminent 临期"
	default:
		return key
	}
}

func toNotifyConfig(row *model.TenantTodoNotify) *NotifyConfigDTO {
	return &NotifyConfigDTO{
		Enabled:             row.Enabled,
		WebhookURL:          row.WebhookURL,
		SecretSet:           strings.TrimSpace(row.Secret) != "",
		PollIntervalMinutes: normalizePollInterval(row.PollIntervalMinutes),
		Levels:              resolveLevels(row),
	}
}

func toNotifyState(row *model.TenantTodoNotify) *NotifyStateDTO {
	st := &NotifyStateDTO{
		LastRunOK:     row.LastRunOK,
		LastError:     row.LastError,
		LastSentCount: row.LastSentCount,
	}
	if row.LastRunAt != nil {
		st.LastRunAt = row.LastRunAt.Format("2006-01-02 15:04:05")
	}
	return st
}

func normalizePollInterval(m int) int {
	if m < 1 {
		return 1
	}
	if m > 24*60 {
		return 24 * 60
	}
	return m
}

func normalizeLeadMinutes(m int) int {
	if m < 0 {
		return 0
	}
	if m > 7*24*60 {
		return 7 * 24 * 60
	}
	return m
}

func priorityLabel(p string) string {
	switch p {
	case model.TodoPriorityHigh:
		return "高"
	case model.TodoPriorityLow:
		return "低"
	default:
		return "普通"
	}
}

func parseNotifiedMap(raw string) map[string]string {
	out := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	if out == nil {
		out = map[string]string{}
	}
	return out
}

func marshalNotified(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func pruneNotified(m map[string]string, keepDays int) map[string]string {
	if keepDays <= 0 {
		keepDays = 60
	}
	cut := time.Now().AddDate(0, 0, -keepDays).Format("2006-01-02")
	out := map[string]string{}
	for k, v := range m {
		day := v
		if day == "" {
			if i := strings.LastIndex(k, ":"); i >= 0 {
				day = k[i+1:]
			}
		}
		if len(day) >= 10 {
			day = day[:10]
		}
		if day >= cut {
			out[k] = v
		}
	}
	return out
}
