// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package util provides shared utility functions across plugins.
package util

import "strings"

// ToLowerCamelCase converts PascalCase to camelCase.
func ToLowerCamelCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// ExtractInnerType extracts the inner type from a generic or array type.
// For example: "List<User>" returns "User", "User[]" returns "User".
func ExtractInnerType(t string) string {
	if strings.HasSuffix(t, "[]") {
		return strings.TrimSuffix(t, "[]")
	}

	start := strings.Index(t, "<")
	end := strings.LastIndex(t, ">")
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(t[start+1 : end])
	}

	return t
}
