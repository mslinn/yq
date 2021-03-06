package yqlib

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/mikefarah/yq/v4/test"
	yaml "gopkg.in/yaml.v3"
)

type expressionScenario struct {
	description           string
	subdescription        string
	document              string
	expression            string
	expected              []string
	skipDoc               bool
	dontFormatInputForDoc bool // dont format input doc for documentation generation
}

func testScenario(t *testing.T, s *expressionScenario) {
	var results *list.List
	var err error

	node, err := treeCreator.ParsePath(s.expression)
	if err != nil {
		t.Error(fmt.Errorf("Error parsing expression %v of %v: %v", s.expression, s.description, err))
		return
	}
	inputs := list.New()

	if s.document != "" {
		inputs, err = readDocuments(strings.NewReader(s.document), "sample.yml", 0)
		if err != nil {
			t.Error(err, s.document, s.expression)
			return
		}
	} else {
		candidateNode := &CandidateNode{
			Document:  0,
			Filename:  "",
			Node:      &yaml.Node{Tag: "!!null"},
			FileIndex: 0,
		}
		inputs.PushBack(candidateNode)

	}

	results, err = treeNavigator.GetMatchingNodes(inputs, node)

	if err != nil {
		t.Error(fmt.Errorf("%v: %v", err, s.expression))
		return
	}
	test.AssertResultComplexWithContext(t, s.expected, resultsToString(results), fmt.Sprintf("exp: %v\ndoc: %v", s.expression, s.document))
}

func resultsToString(results *list.List) []string {
	var pretty []string = make([]string, 0)
	for el := results.Front(); el != nil; el = el.Next() {
		n := el.Value.(*CandidateNode)
		pretty = append(pretty, NodeToString(n))
	}
	return pretty
}

func writeOrPanic(w *bufio.Writer, text string) {
	_, err := w.WriteString(text)
	if err != nil {
		panic(err)
	}
}

func copyFromHeader(title string, out *os.File) error {
	source := fmt.Sprintf("doc/headers/%v.md", title)
	_, err := os.Stat(source)
	if os.IsNotExist(err) {
		return nil
	}
	in, err := os.Open(source) // nolint gosec
	if err != nil {
		return err
	}
	defer safelyCloseFile(in)
	_, err = io.Copy(out, in)
	return err
}

func formatYaml(yaml string) string {
	var output bytes.Buffer
	printer := NewPrinter(bufio.NewWriter(&output), false, true, false, 2, true)

	node, err := treeCreator.ParsePath(".. style= \"\"")
	if err != nil {
		panic(err)
	}
	streamEvaluator := NewStreamEvaluator()
	err = streamEvaluator.Evaluate("sample.yaml", strings.NewReader(yaml), node, printer)
	if err != nil {
		panic(err)
	}
	return output.String()
}

func documentScenarios(t *testing.T, title string, scenarios []expressionScenario) {
	f, err := os.Create(fmt.Sprintf("doc/%v.md", title))

	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()

	err = copyFromHeader(title, f)
	if err != nil {
		t.Error(err)
		return
	}

	w := bufio.NewWriter(f)
	writeOrPanic(w, "\n")

	for _, s := range scenarios {
		if !s.skipDoc {

			writeOrPanic(w, fmt.Sprintf("## %v\n", s.description))

			if s.subdescription != "" {
				writeOrPanic(w, s.subdescription)
				writeOrPanic(w, "\n\n")
			}
			formattedDoc := ""
			if s.document != "" {
				if s.dontFormatInputForDoc {
					formattedDoc = s.document + "\n"
				} else {
					formattedDoc = formatYaml(s.document)
				}
				//TODO: pretty here
				writeOrPanic(w, "Given a sample.yml file of:\n")

				writeOrPanic(w, fmt.Sprintf("```yaml\n%v```\n", formattedDoc))
				writeOrPanic(w, "then\n")
				if s.expression != "" {
					writeOrPanic(w, fmt.Sprintf("```bash\nyq eval '%v' sample.yml\n```\n", s.expression))
				} else {
					writeOrPanic(w, "```bash\nyq eval sample.yml\n```\n")
				}
			} else {
				writeOrPanic(w, "Running\n")
				writeOrPanic(w, fmt.Sprintf("```bash\nyq eval --null-input '%v'\n```\n", s.expression))
			}

			writeOrPanic(w, "will output\n")

			var output bytes.Buffer
			var err error
			printer := NewPrinter(bufio.NewWriter(&output), false, true, false, 2, true)
			streamEvaluator := NewStreamEvaluator()

			if s.document != "" {
				node, err := treeCreator.ParsePath(s.expression)
				if err != nil {
					t.Error(err, s.expression)
				}
				err = streamEvaluator.Evaluate("sample.yaml", strings.NewReader(formattedDoc), node, printer)

				if err != nil {
					t.Error(err, s.expression)
				}
			} else {
				err = streamEvaluator.EvaluateNew(s.expression, printer)
				if err != nil {
					t.Error(err, s.expression)
				}
			}

			writeOrPanic(w, fmt.Sprintf("```yaml\n%v```\n\n", output.String()))

		}

	}
	w.Flush()
}
