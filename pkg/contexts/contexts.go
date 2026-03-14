// Package contexts provides the Contexts & Steps workflow system for
// SignalWire AI agents.
//
// Instead of a single flat prompt, agents can define structured Contexts
// (conversation flows) containing ordered Steps (sequential stages). Each
// step carries its own prompt, completion criteria, function restrictions,
// and navigation rules. The builder serialises the whole tree into the
// map[string]any format expected by the SWML AI verb.
package contexts

import (
	"errors"
	"fmt"
	"strings"
)

// Limits guard against unreasonable configurations.
const (
	MaxContexts       = 50
	MaxStepsPerContext = 100
)

// ---------------------------------------------------------------------------
// GatherQuestion
// ---------------------------------------------------------------------------

// GatherQuestionOption is a functional option applied to a GatherQuestion.
type GatherQuestionOption func(*GatherQuestion)

// WithType sets the JSON-schema type for the answer (default "string").
func WithType(t string) GatherQuestionOption {
	return func(q *GatherQuestion) { q.Type = t }
}

// WithConfirm sets whether the model must confirm the answer with the user.
func WithConfirm(c bool) GatherQuestionOption {
	return func(q *GatherQuestion) { q.Confirm = c }
}

// WithPrompt sets extra instruction text appended for this question.
func WithPrompt(p string) GatherQuestionOption {
	return func(q *GatherQuestion) { q.Prompt = p }
}

// WithFunctions sets additional function names visible for this question.
func WithFunctions(f []string) GatherQuestionOption {
	return func(q *GatherQuestion) { q.Functions = f }
}

// GatherQuestion represents a single question in a gather_info configuration.
type GatherQuestion struct {
	Key       string
	Question  string
	Type      string   // default "string"
	Confirm   bool
	Prompt    string   // optional
	Functions []string // optional
}

// ToMap serialises the question to the SWML map format.
func (q *GatherQuestion) ToMap() map[string]any {
	m := map[string]any{
		"key":      q.Key,
		"question": q.Question,
	}
	if q.Type != "" && q.Type != "string" {
		m["type"] = q.Type
	}
	if q.Confirm {
		m["confirm"] = true
	}
	if q.Prompt != "" {
		m["prompt"] = q.Prompt
	}
	if len(q.Functions) > 0 {
		m["functions"] = q.Functions
	}
	return m
}

// ---------------------------------------------------------------------------
// GatherInfo
// ---------------------------------------------------------------------------

// GatherInfo configures information gathering for a step.
type GatherInfo struct {
	OutputKey        string
	CompletionAction string
	Prompt           string
	Questions        []GatherQuestion
}

// AddQuestion appends a question and returns the GatherInfo for chaining.
func (g *GatherInfo) AddQuestion(key, question string, opts ...GatherQuestionOption) *GatherInfo {
	q := GatherQuestion{
		Key:      key,
		Question: question,
		Type:     "string",
	}
	for _, o := range opts {
		o(&q)
	}
	g.Questions = append(g.Questions, q)
	return g
}

// ToMap serialises to the SWML map format.
func (g *GatherInfo) ToMap() map[string]any {
	qs := make([]map[string]any, len(g.Questions))
	for i := range g.Questions {
		qs[i] = g.Questions[i].ToMap()
	}
	m := map[string]any{
		"questions": qs,
	}
	if g.Prompt != "" {
		m["prompt"] = g.Prompt
	}
	if g.OutputKey != "" {
		m["output_key"] = g.OutputKey
	}
	if g.CompletionAction != "" {
		m["completion_action"] = g.CompletionAction
	}
	return m
}

// ---------------------------------------------------------------------------
// Step
// ---------------------------------------------------------------------------

// Step represents a single step within a context. All setter methods return
// *Step so they can be chained.
type Step struct {
	name               string
	text               string
	sections           []map[string]any
	stepCriteria       string
	functions          any // string "none" or []string
	validSteps         []string
	validContexts      []string
	isEnd              bool
	skipUserTurn       bool
	skipToNextStep     bool
	gatherInfo         *GatherInfo
	resetSystemPrompt  string
	resetUserPrompt    string
	resetConsolidate   *bool
	resetFullReset     *bool
}

// Name returns the step's name.
func (s *Step) Name() string { return s.name }

// SetText sets the step's prompt text directly.
func (s *Step) SetText(text string) *Step {
	s.text = text
	return s
}

// AddSection adds a POM section to the step.
func (s *Step) AddSection(title, body string) *Step {
	s.sections = append(s.sections, map[string]any{"title": title, "body": body})
	return s
}

// AddBullets adds a POM section with bullet points.
func (s *Step) AddBullets(title string, bullets []string) *Step {
	s.sections = append(s.sections, map[string]any{"title": title, "bullets": bullets})
	return s
}

// SetStepCriteria sets the criteria for determining when this step is complete.
func (s *Step) SetStepCriteria(criteria string) *Step {
	s.stepCriteria = criteria
	return s
}

// SetFunctions sets which functions are available in this step.
// Accepts the string "none" to disable all functions, or a []string of names.
func (s *Step) SetFunctions(functions any) *Step {
	s.functions = functions
	return s
}

// SetValidSteps sets which steps can be navigated to from this step.
func (s *Step) SetValidSteps(steps []string) *Step {
	s.validSteps = steps
	return s
}

// SetValidContexts sets which contexts can be navigated to from this step.
func (s *Step) SetValidContexts(contexts []string) *Step {
	s.validContexts = contexts
	return s
}

// SetEnd sets whether the conversation should end after this step.
func (s *Step) SetEnd(end bool) *Step {
	s.isEnd = end
	return s
}

// SetSkipUserTurn sets whether to skip waiting for user input after this step.
func (s *Step) SetSkipUserTurn(skip bool) *Step {
	s.skipUserTurn = skip
	return s
}

// SetSkipToNextStep sets whether to automatically advance to the next step.
func (s *Step) SetSkipToNextStep(skip bool) *Step {
	s.skipToNextStep = skip
	return s
}

// SetGatherInfo enables info gathering for this step and returns the
// GatherInfo so that questions can be added to it directly.
func (s *Step) SetGatherInfo(outputKey, completionAction, prompt string) *GatherInfo {
	s.gatherInfo = &GatherInfo{
		OutputKey:        outputKey,
		CompletionAction: completionAction,
		Prompt:           prompt,
	}
	return s.gatherInfo
}

// AddGatherQuestion adds a question to this step's gather_info. SetGatherInfo
// must be called first. Returns the Step for chaining.
func (s *Step) AddGatherQuestion(key, question string, opts ...GatherQuestionOption) *Step {
	if s.gatherInfo == nil {
		// Silently initialise so callers are not forced into ordering.
		s.gatherInfo = &GatherInfo{}
	}
	s.gatherInfo.AddQuestion(key, question, opts...)
	return s
}

// ClearSections removes all POM sections and direct text from this step.
func (s *Step) ClearSections() *Step {
	s.sections = nil
	s.text = ""
	return s
}

// SetResetSystemPrompt sets the system prompt for context switching.
func (s *Step) SetResetSystemPrompt(prompt string) *Step {
	s.resetSystemPrompt = prompt
	return s
}

// SetResetUserPrompt sets the user prompt for context switching.
func (s *Step) SetResetUserPrompt(prompt string) *Step {
	s.resetUserPrompt = prompt
	return s
}

// SetResetConsolidate sets whether to consolidate conversation on context switch.
func (s *Step) SetResetConsolidate(consolidate bool) *Step {
	s.resetConsolidate = &consolidate
	return s
}

// SetResetFullReset sets whether to do a full reset on context switch.
func (s *Step) SetResetFullReset(fullReset bool) *Step {
	s.resetFullReset = &fullReset
	return s
}

// renderText produces the prompt string for the step.
func (s *Step) renderText() string {
	if s.text != "" {
		return s.text
	}
	if len(s.sections) == 0 {
		return ""
	}
	var parts []string
	for _, sec := range s.sections {
		title, _ := sec["title"].(string)
		if bullets, ok := sec["bullets"].([]string); ok {
			parts = append(parts, fmt.Sprintf("## %s", title))
			for _, b := range bullets {
				parts = append(parts, fmt.Sprintf("- %s", b))
			}
		} else {
			body, _ := sec["body"].(string)
			parts = append(parts, fmt.Sprintf("## %s", title))
			parts = append(parts, body)
		}
		parts = append(parts, "") // blank line for spacing
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// ToMap serialises the step to the SWML map format.
func (s *Step) ToMap() map[string]any {
	m := map[string]any{
		"name": s.name,
		"text": s.renderText(),
	}

	if s.stepCriteria != "" {
		m["step_criteria"] = s.stepCriteria
	}
	if s.functions != nil {
		m["functions"] = s.functions
	}
	if s.validSteps != nil {
		m["valid_steps"] = s.validSteps
	}
	if s.validContexts != nil {
		m["valid_contexts"] = s.validContexts
	}
	if s.isEnd {
		m["end"] = true
	}
	if s.skipUserTurn {
		m["skip_user_turn"] = true
	}
	if s.skipToNextStep {
		m["skip_to_next_step"] = true
	}

	// Build reset object if any reset field is set.
	reset := map[string]any{}
	if s.resetSystemPrompt != "" {
		reset["system_prompt"] = s.resetSystemPrompt
	}
	if s.resetUserPrompt != "" {
		reset["user_prompt"] = s.resetUserPrompt
	}
	if s.resetConsolidate != nil && *s.resetConsolidate {
		reset["consolidate"] = true
	}
	if s.resetFullReset != nil && *s.resetFullReset {
		reset["full_reset"] = true
	}
	if len(reset) > 0 {
		m["reset"] = reset
	}

	if s.gatherInfo != nil && len(s.gatherInfo.Questions) > 0 {
		m["gather_info"] = s.gatherInfo.ToMap()
	}

	return m
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

// Context represents a single context containing ordered steps.
// All setter methods return *Context for chaining.
type Context struct {
	name           string
	steps          []*Step // ordered
	stepMap        map[string]*Step
	validContexts  []string
	validSteps     []string
	postPrompt     string
	systemPrompt   string
	prompt         string
	consolidate    *bool
	fullReset      *bool
	userPrompt     string
	isolated       bool
	sections       []map[string]any
	systemSections []map[string]any
	enterFillers   map[string][]string
	exitFillers    map[string][]string
}

// newContext creates a Context with the given name.
func newContext(name string) *Context {
	return &Context{
		name:    name,
		stepMap: make(map[string]*Step),
	}
}

// Name returns the context's name.
func (c *Context) Name() string { return c.name }

// AddStep creates a new step, appends it to the ordered list, stores it in
// the lookup map, and returns the Step for further configuration.
func (c *Context) AddStep(name string) *Step {
	s := &Step{name: name}
	c.steps = append(c.steps, s)
	c.stepMap[name] = s
	return s
}

// GetStep returns the step with the given name, or nil if not found.
func (c *Context) GetStep(name string) *Step {
	return c.stepMap[name]
}

// RemoveStep removes a step by name.
func (c *Context) RemoveStep(name string) {
	if _, ok := c.stepMap[name]; !ok {
		return
	}
	delete(c.stepMap, name)
	for i, s := range c.steps {
		if s.name == name {
			c.steps = append(c.steps[:i], c.steps[i+1:]...)
			break
		}
	}
}

// MoveStep moves an existing step to the given position (0-based index).
func (c *Context) MoveStep(name string, position int) {
	idx := -1
	for i, s := range c.steps {
		if s.name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	step := c.steps[idx]
	// Remove from current position.
	c.steps = append(c.steps[:idx], c.steps[idx+1:]...)
	// Clamp position.
	if position < 0 {
		position = 0
	}
	if position > len(c.steps) {
		position = len(c.steps)
	}
	// Insert at new position.
	c.steps = append(c.steps[:position], append([]*Step{step}, c.steps[position:]...)...)
}

// SetValidContexts sets which contexts can be navigated to from this context.
func (c *Context) SetValidContexts(ctxs []string) *Context {
	c.validContexts = ctxs
	return c
}

// SetValidSteps sets which steps can be navigated to from any step in this context.
func (c *Context) SetValidSteps(steps []string) *Context {
	c.validSteps = steps
	return c
}

// SetPostPrompt sets the post-prompt override for this context.
func (c *Context) SetPostPrompt(prompt string) *Context {
	c.postPrompt = prompt
	return c
}

// SetSystemPrompt sets the system prompt for context switching.
func (c *Context) SetSystemPrompt(prompt string) *Context {
	c.systemPrompt = prompt
	return c
}

// SetPrompt sets the context's prompt text directly.
func (c *Context) SetPrompt(prompt string) *Context {
	c.prompt = prompt
	return c
}

// SetConsolidate sets whether to consolidate conversation history on entry.
func (c *Context) SetConsolidate(consolidate bool) *Context {
	c.consolidate = &consolidate
	return c
}

// SetFullReset sets whether to do a full reset when entering this context.
func (c *Context) SetFullReset(fullReset bool) *Context {
	c.fullReset = &fullReset
	return c
}

// SetUserPrompt sets the user prompt to inject when entering this context.
func (c *Context) SetUserPrompt(prompt string) *Context {
	c.userPrompt = prompt
	return c
}

// SetIsolated sets whether to truncate conversation history on entry.
func (c *Context) SetIsolated(isolated bool) *Context {
	c.isolated = isolated
	return c
}

// AddSection adds a POM section to the context prompt.
func (c *Context) AddSection(title, body string) *Context {
	c.sections = append(c.sections, map[string]any{"title": title, "body": body})
	return c
}

// AddBullets adds a POM section with bullet points to the context prompt.
func (c *Context) AddBullets(title string, bullets []string) *Context {
	c.sections = append(c.sections, map[string]any{"title": title, "bullets": bullets})
	return c
}

// AddSystemSection adds a POM section to the system prompt.
func (c *Context) AddSystemSection(title, body string) *Context {
	c.systemSections = append(c.systemSections, map[string]any{"title": title, "body": body})
	return c
}

// AddSystemBullets adds a POM section with bullet points to the system prompt.
func (c *Context) AddSystemBullets(title string, bullets []string) *Context {
	c.systemSections = append(c.systemSections, map[string]any{"title": title, "bullets": bullets})
	return c
}

// SetEnterFillers sets all enter fillers at once.
func (c *Context) SetEnterFillers(fillers map[string][]string) *Context {
	c.enterFillers = fillers
	return c
}

// SetExitFillers sets all exit fillers at once.
func (c *Context) SetExitFillers(fillers map[string][]string) *Context {
	c.exitFillers = fillers
	return c
}

// AddEnterFiller adds enter fillers for a specific language code.
func (c *Context) AddEnterFiller(langCode string, fillers []string) *Context {
	if c.enterFillers == nil {
		c.enterFillers = make(map[string][]string)
	}
	c.enterFillers[langCode] = fillers
	return c
}

// AddExitFiller adds exit fillers for a specific language code.
func (c *Context) AddExitFiller(langCode string, fillers []string) *Context {
	if c.exitFillers == nil {
		c.exitFillers = make(map[string][]string)
	}
	c.exitFillers[langCode] = fillers
	return c
}

// renderSections converts a slice of POM section maps into a markdown string.
func renderSections(sections []map[string]any) string {
	if len(sections) == 0 {
		return ""
	}
	var parts []string
	for _, sec := range sections {
		title, _ := sec["title"].(string)
		if bullets, ok := sec["bullets"].([]string); ok {
			parts = append(parts, fmt.Sprintf("## %s", title))
			for _, b := range bullets {
				parts = append(parts, fmt.Sprintf("- %s", b))
			}
		} else {
			body, _ := sec["body"].(string)
			parts = append(parts, fmt.Sprintf("## %s", title))
			parts = append(parts, body)
		}
		parts = append(parts, "")
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// ToMap serialises the context to the SWML map format.
func (c *Context) ToMap() map[string]any {
	m := map[string]any{}

	// Steps (ordered).
	stepList := make([]map[string]any, len(c.steps))
	for i, s := range c.steps {
		stepList[i] = s.ToMap()
	}
	m["steps"] = stepList

	if c.validContexts != nil {
		m["valid_contexts"] = c.validContexts
	}
	if c.validSteps != nil {
		m["valid_steps"] = c.validSteps
	}
	if c.postPrompt != "" {
		m["post_prompt"] = c.postPrompt
	}

	// System prompt: POM sections take precedence over raw string.
	if len(c.systemSections) > 0 {
		m["system_prompt"] = renderSections(c.systemSections)
	} else if c.systemPrompt != "" {
		m["system_prompt"] = c.systemPrompt
	}

	if c.consolidate != nil && *c.consolidate {
		m["consolidate"] = true
	}
	if c.fullReset != nil && *c.fullReset {
		m["full_reset"] = true
	}
	if c.userPrompt != "" {
		m["user_prompt"] = c.userPrompt
	}
	if c.isolated {
		m["isolated"] = true
	}

	// Context prompt: POM sections produce "pom" key, raw string uses "prompt".
	if len(c.sections) > 0 {
		m["pom"] = c.sections
	} else if c.prompt != "" {
		m["prompt"] = c.prompt
	}

	if c.enterFillers != nil {
		m["enter_fillers"] = c.enterFillers
	}
	if c.exitFillers != nil {
		m["exit_fillers"] = c.exitFillers
	}

	return m
}

// ---------------------------------------------------------------------------
// ContextBuilder
// ---------------------------------------------------------------------------

// ContextBuilder is the top-level builder for creating a set of contexts.
type ContextBuilder struct {
	contexts   []*Context // ordered
	contextMap map[string]*Context
}

// NewContextBuilder creates a new empty ContextBuilder.
func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{
		contextMap: make(map[string]*Context),
	}
}

// AddContext creates a new context with the given name and returns it.
func (cb *ContextBuilder) AddContext(name string) *Context {
	ctx := newContext(name)
	cb.contexts = append(cb.contexts, ctx)
	cb.contextMap[name] = ctx
	return ctx
}

// GetContext returns the context with the given name, or nil if not found.
func (cb *ContextBuilder) GetContext(name string) *Context {
	return cb.contextMap[name]
}

// Validate checks the builder configuration for common errors:
//   - At least one context must be defined.
//   - A single context must be named "default".
//   - Every context must contain at least one step.
//   - Every step must have a name.
func (cb *ContextBuilder) Validate() error {
	if len(cb.contexts) == 0 {
		return errors.New("at least one context must be defined")
	}
	if len(cb.contexts) == 1 && cb.contexts[0].name != "default" {
		return fmt.Errorf("when using a single context, it must be named 'default' (got %q)", cb.contexts[0].name)
	}
	for _, ctx := range cb.contexts {
		if len(ctx.steps) == 0 {
			return fmt.Errorf("context %q must have at least one step", ctx.name)
		}
		for _, s := range ctx.steps {
			if s.name == "" {
				return fmt.Errorf("all steps in context %q must have a name", ctx.name)
			}
		}
	}
	return nil
}

// ToMap serialises all contexts to the SWML map format. It calls Validate
// first and returns an error if validation fails.
func (cb *ContextBuilder) ToMap() (map[string]any, error) {
	if err := cb.Validate(); err != nil {
		return nil, err
	}
	m := make(map[string]any, len(cb.contexts))
	for _, ctx := range cb.contexts {
		m[ctx.name] = ctx.ToMap()
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// CreateSimpleContext creates a standalone Context. If name is empty it
// defaults to "default".
func CreateSimpleContext(name string) *Context {
	if name == "" {
		name = "default"
	}
	return newContext(name)
}
