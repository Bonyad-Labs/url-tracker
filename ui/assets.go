// Package ui contains embedded assets and UI components.
package ui

import "encoding/base64"

// A simple 16x16 black dot icon encoded as PNG to serve as a placeholder menu bar icon if image generation fails
// Or better yet, just a small transparent icon with a black square.
// This is a base64 of a 16x16 black square PNG.
var AppIconBase64 = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAIAAACQkWg2AAAAAXNSR0IArs4c6QAAAERlWElmTU0AKgAAAAgAAYdpAAQAAAABAAAAGgAAAAAAA6ABAAMAAAABAAEAAKACAAQAAAABAAAAEKADAAQAAAABAAAAEAAAAAA0VXHyAAACCElEQVQoFU1STU/bQBDdL5tUal1AgA+UREJtLIUeEpJgLpGAQ0T/SznQY9s/QH9DW6kcE/InonIgB3ovhygKBJBoUSKg1LG9fbNLVGxLM7vz5s2bGfOl7NIz76lmTKep1kwIDp9pzbkgQw5njGud4trznitCA4g7LrgBI46DgcLSCZnWGQ6HoKGH47XWGDAAaXhMkKKWhylDb9GkYDyOkiQVQriuA+yDIsNCdTlHgqlv9Wq2sbGZzWbPzwftdjuOE8bQmFYKycSMsuCicrb6u93dcG3t/v7PSmHlw/uP6Glx8UX+VT5JYlJGIrWcm5+DDyW1Wi2Xy+192gNxq9UKgny9Xq9Wq2EY3t7ddbtdKSSQgioxDd1A93q9UrFUqZTXw/XjH8dBEHz++mUwOJudnjYzB1ZL3/eRgaKe5xWLpf39b4i92d4uVypPMhlU6J+eNg+ajuNAfRT95csvlzNTU0maQuDbnR0l5czsjD/v39ze/Dw5OWg2r35dSSknC2FywV+gjaEjzjudoyiKtja3Li8uhRSNRqPf7zuOiyDU44N8ZceEIzaNqR91Or+vr8urq98PD6HedV0KTcbIMdPC68LDWsjQPxLH8XgcY3FQ8ihkwqiASoR69CiFRpSZOt1O9FiUFqSNUuxHCMzOgAg6ofvPKUajEbVsIAYKF80AbZk0JmLbRRSj/wfXZ+4y7rbkvAAAAABJRU5ErkJggg=="

func GetAppIcon() []byte {
	b, _ := base64.StdEncoding.DecodeString(AppIconBase64)
	return b
}
