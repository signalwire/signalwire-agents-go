// Package swml provides the SWML (SignalWire Markup Language) document model,
// builder, and rendering for the SignalWire AI platform.
//
// SWML documents define call flows, AI agent behavior, and telephony operations.
// The SignalWire platform fetches SWML from agents and executes it.
package swml

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Document represents a complete SWML document with version and sections.
type Document struct {
	mu       sync.RWMutex
	Version  string
	Sections map[string][]Verb
}

// Verb represents a single SWML verb (instruction) as a key-value pair.
// Example: {"play": {"url": "https://example.com/audio.mp3"}}
type Verb map[string]any

// NewDocument creates a new empty SWML document with default version.
func NewDocument() *Document {
	return &Document{
		Version: "1.0.0",
		Sections: map[string][]Verb{
			"main": {},
		},
	}
}

// Reset clears all sections and recreates the default "main" section.
func (d *Document) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Sections = map[string][]Verb{
		"main": {},
	}
}

// AddSection creates a new named section in the document.
// Returns false if the section already exists.
func (d *Document) AddSection(name string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.Sections[name]; exists {
		return false
	}
	d.Sections[name] = []Verb{}
	return true
}

// HasSection returns whether a section exists in the document.
func (d *Document) HasSection(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, exists := d.Sections[name]
	return exists
}

// AddVerb appends a verb to the "main" section.
// Returns an error if the verb name is empty.
func (d *Document) AddVerb(verbName string, config any) error {
	return d.AddVerbToSection("main", verbName, config)
}

// AddVerbToSection appends a verb to a named section.
// Creates the section if it doesn't exist.
func (d *Document) AddVerbToSection(section, verbName string, config any) error {
	if verbName == "" {
		return fmt.Errorf("verb name cannot be empty")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.Sections[section]; !exists {
		d.Sections[section] = []Verb{}
	}
	verb := Verb{verbName: config}
	d.Sections[section] = append(d.Sections[section], verb)
	return nil
}

// GetVerbs returns the verbs in a named section.
// Returns nil if the section doesn't exist.
func (d *Document) GetVerbs(section string) []Verb {
	d.mu.RLock()
	defer d.mu.RUnlock()
	verbs, exists := d.Sections[section]
	if !exists {
		return nil
	}
	result := make([]Verb, len(verbs))
	copy(result, verbs)
	return result
}

// ToMap returns the document as a nested map suitable for JSON serialization.
func (d *Document) ToMap() map[string]any {
	d.mu.RLock()
	defer d.mu.RUnlock()

	sections := make(map[string]any, len(d.Sections))
	for name, verbs := range d.Sections {
		verbList := make([]any, len(verbs))
		for i, v := range verbs {
			verbList[i] = map[string]any(v)
		}
		sections[name] = verbList
	}

	return map[string]any{
		"version":  d.Version,
		"sections": sections,
	}
}

// Render serializes the document to a JSON string.
func (d *Document) Render() (string, error) {
	data, err := json.Marshal(d.ToMap())
	if err != nil {
		return "", fmt.Errorf("failed to render SWML document: %w", err)
	}
	return string(data), nil
}

// RenderPretty serializes the document to an indented JSON string.
func (d *Document) RenderPretty() (string, error) {
	data, err := json.MarshalIndent(d.ToMap(), "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to render SWML document: %w", err)
	}
	return string(data), nil
}

// MarshalJSON implements the json.Marshaler interface.
func (d *Document) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.ToMap())
}
