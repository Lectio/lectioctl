package main

import (
	"errors"

	"gopkg.in/jdkato/prose.v2"
)

type naturalLanguageProcessor struct{}

func makeDefaultNLP() *naturalLanguageProcessor {
	return new(naturalLanguageProcessor)
}

func (nlp naturalLanguageProcessor) FirstSentence(source string) (string, error) {
	content, proseErr := prose.NewDocument(source)
	if proseErr != nil {
		return "", proseErr
	}

	sentences := content.Sentences()
	if len(sentences) > 0 {
		return sentences[0].Text, nil
	}
	return "", errors.New("Unable to find any sentences in the body")
}
