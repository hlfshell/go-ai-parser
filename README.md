# arkaine-parser

**arkaine-parser** is a Go module for robustly parsing stochastic, structured, or semi-structured outputs from Large Language Models (LLMs) into clean, agent-ready data structures. It is inspired by the original [arkaine parser](https://arkaine.dev) and designed for easy integration into any Go AI or agentic workflow.

**Import Path:**
```
import "github.com/hlfshell/go-arkaine-parser"
```
**Go Package Name:**
```
package arkaineparser
```

> The GitHub repo and Go module is `go-arkaine-parser`, but the package name used in your Go code is `arkaineparser`. All code examples below follow this convention.

## Features
- **Flexible label definitions** 📝: Case-insensitive, multi-word, custom separators
- **Multiline and nested value support** 📄
- **JSON field extraction and error handling** 📊
- **Required and dependency validation** 🚫
- **Block-based parsing for multi-entry outputs** 📚
- **Asset-driven, comprehensive tests** 🧪
- **Idiomatic Go API and easy import** 📦

## Installation

```sh
go get github.com/hlfshell/arkaine-parser
```

## Usage Examples

### Labels

Labels convey to the parser what to expect from the LLM output and guide parsing.

Labels have the following properties:

- **Name**: (string) The label name to match (case-insensitive, multi-word allowed, and whitespace-insensitive).
- **Required**: (bool) If true, this label must be present.
- **RequiredWith**: ([]string) List of label names that must also be present if this label is present.
- **IsJSON**: (bool) If true, the label value is parsed as JSON.
- **IsBlockStart**: (bool) If true, this label marks the start of a new block for block parsing (see ParseBlock)

**Label matching rules:**
- Labels are matched at the start of a line, case-insensitive, and allow multi-word labels (e.g., `Action Input`).
- Flexible separators are supported: colon (`:`), tilde (`~`), dash (`-`), etc. For example, `Action Input ~ value` is valid.
- Unknown labels in LLM output are ignored. If a label is defined but not present in the output, its value will be `""` (empty string) in the result.

### Parse

Parse is when you expect a single output from a singular LLM response. Before parsing, the input is automatically cleaned: any markdown code blocks (```...```) and inline code (`...`) will be removed for robust parsing.

For instance, let's assume that the LLM is being asked which tool to call, and thus we want it to produce its reasoning, what function it called, and what parameters it should be passed (JSON formatted):

```go
package main

import (
    "fmt"
    "github.com/hlfshell/go-arkaine-parser"
)

func main() {
    // Define your labels
    labels := []arkaineparser.Label{
        {Name: "Reasoning", IsBlockStart: true},
        {Name: "Function"},
        {Name: "Parameters", IsJSON: true},
    }
    parser, err := arkaineparser.NewParser(labels)
    if err != nil {
        panic(err)
    }
    
    // Call your LLM/agent and, assuming its output
    // is in text

    text := `
Reasoning: I need to process some files.
Function: process_data
Parameters: {"input_files": ["a.txt", "b.txt"]}
`

    result, errs := parser.Parse(text)
    fmt.Println("Result:", result)
    fmt.Println("Errors:", errs)

    // Assuming errs is nil/empty, you can
    // access your parsed block
    if len(errs) == 0 {
        fmt.Println("Reasoning:", result["Reasoning"])
        fmt.Println("Function:", result["Function"])
        fmt.Println("Parameters:", result["Parameters"])
    }

    // ...call your tool, etc
}
```

### ParseBlocks

ParseBlocks is when you expect to have an unknown number of outputs from a singular LLM response.

```go
package main

import (
    "fmt"
    "github.com/hlfshell/go-arkaine-parser"
)

func main() {
    // Define your labels
    labels := []arkaineparser.Label{
        {Name: "Task", IsBlockStart: true},
        {Name: "Thought"},
        {Name: "Result"},
    }
    parser, err := arkaineparser.NewParser(labels)
    if err != nil {
        panic(err)
    }
    
    // Call your LLM/agent and, assuming its output
    // is in text

    blocks, errs := parser.ParseBlocks(text)
    fmt.Println("Blocks:", blocks)
    fmt.Println("Errors:", errors)

    // Assuming errs is nil/empty, you can
    // access your parsed blocksv
    var summary str
    var classification str
    if len(errs) == 0 {
        for _, block := range blocks {
            if block["Task"] == "Summarize" {
                summary = block["Result"]
            } else if block["Task"] == "Classify" {
                classification = block["Result"]
            }
        }
    }

    // ...
}
```

---

### Agentic Example: Sentiment Classification

This example demonstrates how to use `arkaine-parser` in a real agent workflow, closely following best practices for agentic LLM prompting and structured output parsing. It mirrors the agentic Python example, but is idiomatic Go and heavily commented.

#### Example LLM Prompt

```markdown
**You are an advanced classification agent tasked with accurately assigning sentiment labels to a given body of text.** Your classification must be based strictly on a predefined set of labels, definitions, and examples. Your goal is to ensure precise and justifiable labeling while avoiding misclassification.  

# **Instructions:**  
1. **Analyze** the provided text carefully, identifying key sentiment indicators (e.g., tone, word choice, intensity).  
2. **Compare** the text against the label definitions and examples provided.  
3. **Assign** the most appropriate label(s) based on the closest semantic match.  
4. **Justify** your label assignment with a concise explanation, explicitly referencing relevant words or phrases in the text.  
5. **Avoid Misclassification**: If the text does not match any label clearly, return `"Label: None"` with an explanation.  

# **Available Labels:**  
Positive, Negative, Neutral, None

# **Response Format:**  
For each classification, use the following structured output:  

Reason: [Explicit justification referencing words or phrases from the text]
Label: [Positive/Negative/Neutral/None]  

#### **Final Notes:**  
- Be precise in your analysis—avoid overgeneralizing sentiment.  
- If the text is ambiguous or lacks clear sentiment, classify it as `"None"`.  
- Always provide a direct textual reference to support your classification.

**Input:**
{input}

**Output**:
```

#### Code:

```go
package main

import (
    "fmt"
    "github.com/hlfshell/go-arkaine-parser"
)

func main() {
    // Step 1: Define the labels expected in the LLM output.
    // These must match the prompt and LLM output format exactly.
    labels := []arkaineparser.Label{
        {
            Name: "Reason", // Explanation for label assignment
        },
        {
            Name: "Label", // Sentiment label (Positive/Negative/Neutral/None)
        },
    }

    // Step 2: Create the parser instance.
    parser, err := arkaineparser.NewParser(labels)
    if err != nil {
        panic(err)
    }

    // Step 3: We call the LLM, getting a response in llmOutput
    llmOutput := "lorem ipsum"

    // Step 4: Parse the LLM output into a structured map.
    result, errs := parser.Parse(llmOutput)
    if len(errs) != 0 {
        fmt.Println("Parse errors:", errs)
        return
    }

    // Step 5: Use the parsed results in your agent logic.
    // Variable names are short but descriptive, as per project guidelines.
    reason := result["Reason"]
    sentimentLabel := result["Label"]

    fmt.Println("Reason for label:", reason)
    fmt.Println("Assigned label:", sentimentLabel)

    // Example agentic branch: take action based on label.
    switch sentimentLabel {
    case "Positive":
        fmt.Println("Take positive action!")
    case "Negative":
        fmt.Println("Handle negative sentiment.")
    case "Neutral":
        fmt.Println("No action needed.")
    case "None":
        fmt.Println("Ambiguous or no sentiment detected.")
    default:
        fmt.Println("Unknown label.")
    }
}
```


## Error Handling & Return Types

When using `Parse` or `ParseBlocks`, you receive two return values:

- The parsed result (`map[string]interface{}` for `Parse`, or a slice of those for `ParseBlocks`).
- A slice of error strings (`[]string`).

**Error Handling:**
- Errors are returned for missing required labels, missing dependencies (`RequiredWith`), and JSON parsing errors.
- Example errors:
  - `'Function' is required`
  - `'Parameters' requires 'Function'`
  - `JSON error in 'Parameters': invalid character '}' looking for beginning of object key string`
- Always check the `errs` slice before using the parsed results.

**Return Types:**
- Each value in the result map can be:
  - A string (for plain values)
  - A parsed JSON object (for labels marked with `IsJSON`)
  - A slice of values (if the label appears multiple times)
- If a label is defined but not present, its value will be `""` (empty string).
- All label keys in the result are lowercased.

---

## Prompting LLMs for Structured Output

To get the most reliable results from LLMs with `arkaine-parser`, you should prompt the model to output labeled sections that match your label definitions. Here’s how to design your prompts and what the parser expects:

### Label Expectations
- **Each label should appear at the start of a line**, followed by a separator (usually a colon and space, e.g., `Label: value`).
- **Labels are case-insensitive**: `Task:`, `task:`, and `TASK:` are all equivalent.
- **Multiline values**: If a value spans multiple lines, only the first line starts with the label; continuation lines should not start with another label.
- **JSON fields**: If a label is marked as `IsJSON`, instruct the model to output valid JSON for that field.
- **Block start**: If using block parsing, instruct the model to repeat the block start label for each new block.

### Example Prompt for LLM

```
You are an AI agent. When given a task, respond in the following format:

Task: <short task description>
Thought: <Your thoughts on how to accomplish the task>
Result: <result of the task>

If there are multiple tasks, repeat the block for each one, always starting with 'Task:'.
```

### Example LLM Output
```
<Some body of text you insert for the AI to analyze>

Task: Summarize
Thought: This content is an extensive blogpost on the benefits of using skydiving to solve insomnia, discussing...
Result: Skydiving is a great way to solve insomnia.

Task: Classify
Thought: This content is a self-help blogpost
Result: self-help blogpost
```

### Tips for Robust Prompting
- **Explicitly specify the label format** in your prompt (e.g., `Label: value`).
- **Instruct the model to use valid JSON** for any field marked as `IsJSON`.
- **Avoid ambiguous or missing labels**—the parser expects every label to be on its own line.
- **For required fields or dependencies**, make it clear in your prompt that these must always be included.
- **For multiline values**, instruct the model to avoid starting continuation lines with a label name.

By following these guidelines, you ensure that LLM outputs are easy to parse and robustly handled by `arkaine-parser` in your Go projects.

## Testing

- All test inputs and expected outputs are stored as readable files in `assets/`.
- Tests cover mixed case, multiline, JSON/malformed JSON, dependency validation, and block parsing.
- To run tests:
  ```sh
  go test -v ./...
  ```

---

**arkaine-parser** is open source and ready for use in any Go-based AI or agentic project. It is released under the MIT license.