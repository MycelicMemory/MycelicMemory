// Package locomo implements the LoCoMo benchmark for evaluating
// long-term conversational memory in ultrathink.
//
// LoCoMo (Long-term Conversational Memory) is an ACL 2024 benchmark
// that tests LLM agents on question answering and event summarization
// across very long conversation histories (300+ turns, 35+ sessions).
//
// See: https://github.com/snap-research/locomo
package locomo

import "time"

// Dataset represents the full LoCoMo dataset
type Dataset struct {
	Conversations []Conversation `json:"conversations"`
}

// Conversation represents a single LoCoMo conversation between two speakers
type Conversation struct {
	// ID is the unique identifier for this conversation
	ID string `json:"id"`

	// SpeakerA and SpeakerB are the names of the two participants
	SpeakerA string `json:"speaker_a"`
	SpeakerB string `json:"speaker_b"`

	// Personas contains personality traits for each speaker
	// Map from speaker name to list of persona sentences
	Personas map[string][]string `json:"personas"`

	// Sessions contains dialogue turns organized by session
	// Map from "session_N" to list of turns
	Sessions map[string][]Turn `json:"sessions"`

	// SessionDates contains timestamps for each session
	// Map from "session_N_date_time" to timestamp string
	SessionDates map[string]string `json:"session_dates"`

	// QA contains question-answer annotations for evaluation
	QA []QAAnnotation `json:"qa"`

	// Events contains event graph annotations for summarization evaluation
	Events []EventAnnotation `json:"events"`

	// Observations contains generated observations about speakers (optional)
	Observations map[string][]string `json:"observations,omitempty"`

	// Summaries contains session summaries (optional)
	Summaries map[string]string `json:"summaries,omitempty"`
}

// Turn represents a single dialogue turn within a session
type Turn struct {
	// DiaID is the unique dialogue turn identifier (e.g., "dia_1_1")
	DiaID string `json:"dia_id"`

	// Speaker is the name of the speaker for this turn
	Speaker string `json:"speaker"`

	// Content is the text content of the dialogue turn
	Content string `json:"content"`

	// ImageURL is the URL of an image shared in this turn (optional)
	ImageURL string `json:"image_url,omitempty"`

	// ImageCaption is the caption for the shared image (optional)
	ImageCaption string `json:"image_caption,omitempty"`
}

// QAAnnotation represents a question-answer pair for evaluation
type QAAnnotation struct {
	// Question is the evaluation question
	Question string `json:"question"`

	// Answer is the ground truth answer
	Answer string `json:"answer"`

	// Category is the question type: single_hop, multi_hop, temporal,
	// commonsense, or adversarial
	Category QuestionCategory `json:"category"`

	// Evidence contains dialogue IDs that contain the answer
	Evidence []string `json:"evidence,omitempty"`
}

// QuestionCategory represents the type of QA question
type QuestionCategory string

const (
	CategorySingleHop   QuestionCategory = "single_hop"
	CategoryMultiHop    QuestionCategory = "multi_hop"
	CategoryTemporal    QuestionCategory = "temporal"
	CategoryCommonsense QuestionCategory = "commonsense"
	CategoryAdversarial QuestionCategory = "adversarial"
)

// EventAnnotation represents an event graph annotation
type EventAnnotation struct {
	// Speaker is the name of the speaker this event relates to
	Speaker string `json:"speaker"`

	// Session is the session number where this event occurred
	Session int `json:"session"`

	// Event is the description of the significant event
	Event string `json:"event"`

	// Causes lists events that caused this event
	Causes []string `json:"causes,omitempty"`

	// Effects lists events that resulted from this event
	Effects []string `json:"effects,omitempty"`
}

// IngestionResult contains statistics from data ingestion
type IngestionResult struct {
	// ConversationsIngested is the number of conversations processed
	ConversationsIngested int `json:"conversations_ingested"`

	// TotalTurns is the total number of dialogue turns ingested
	TotalTurns int `json:"total_turns"`

	// TotalMemories is the total number of memories created
	TotalMemories int `json:"total_memories"`

	// PersonaMemories is the number of persona memories created
	PersonaMemories int `json:"persona_memories"`

	// TotalQAQuestions is the total number of QA questions available
	TotalQAQuestions int `json:"total_qa_questions"`

	// Duration is how long ingestion took
	Duration time.Duration `json:"duration"`
}

// EvaluationConfig contains configuration for benchmark evaluation
type EvaluationConfig struct {
	// Task is the evaluation task: "qa" or "events"
	Task string `json:"task"`

	// RetrievalStrategy is the retrieval approach to use
	RetrievalStrategy RetrievalStrategy `json:"retrieval_strategy"`

	// TopK is the number of memories to retrieve (for RAG strategies)
	TopK int `json:"top_k"`

	// Category filters QA questions to a specific category (optional)
	Category QuestionCategory `json:"category,omitempty"`

	// ConversationIDs filters to specific conversations (optional)
	ConversationIDs []string `json:"conversation_ids,omitempty"`

	// Verbose enables detailed logging during evaluation
	Verbose bool `json:"verbose"`
}

// RetrievalStrategy represents the memory retrieval approach
type RetrievalStrategy string

const (
	// StrategyDirect uses all conversation memories as context
	StrategyDirect RetrievalStrategy = "direct"

	// StrategyDialogRAG uses semantic search over dialogue turns
	StrategyDialogRAG RetrievalStrategy = "dialog-rag"

	// StrategyObservationRAG uses pre-generated observations
	StrategyObservationRAG RetrievalStrategy = "observation-rag"

	// StrategySummaryRAG uses session summaries
	StrategySummaryRAG RetrievalStrategy = "summary-rag"
)

// QuestionResult contains the result for a single QA question
type QuestionResult struct {
	// ConversationID is the source conversation
	ConversationID string `json:"conversation_id"`

	// Question is the evaluation question
	Question string `json:"question"`

	// Category is the question type
	Category QuestionCategory `json:"category"`

	// GroundTruth is the expected answer
	GroundTruth string `json:"ground_truth"`

	// GeneratedAnswer is the model's answer
	GeneratedAnswer string `json:"generated_answer"`

	// RetrievedMemories is the number of memories used
	RetrievedMemories int `json:"retrieved_memories"`

	// F1 is the token-level F1 score
	F1 float64 `json:"f1"`

	// Precision is the token-level precision
	Precision float64 `json:"precision"`

	// Recall is the token-level recall
	Recall float64 `json:"recall"`

	// EvidenceFound indicates if the evidence memories were retrieved
	EvidenceFound bool `json:"evidence_found"`
}

// Metrics contains aggregated evaluation metrics
type Metrics struct {
	// F1 is the average F1 score
	F1 float64 `json:"f1"`

	// Precision is the average precision
	Precision float64 `json:"precision"`

	// Recall is the average recall
	Recall float64 `json:"recall"`

	// Count is the number of questions evaluated
	Count int `json:"count"`
}

// BenchmarkResults contains complete benchmark evaluation results
type BenchmarkResults struct {
	// Benchmark is the benchmark name ("locomo")
	Benchmark string `json:"benchmark"`

	// Version is the ultrathink version used
	Version string `json:"version"`

	// Timestamp is when the evaluation was run
	Timestamp time.Time `json:"timestamp"`

	// Model is the AI model used (e.g., "ollama/qwen2.5:3b")
	Model string `json:"model"`

	// Strategy is the retrieval strategy used
	Strategy RetrievalStrategy `json:"retrieval_strategy"`

	// Config contains the evaluation configuration
	Config EvaluationConfig `json:"config"`

	// Overall contains aggregate metrics across all questions
	Overall Metrics `json:"overall"`

	// Categories contains metrics broken down by question category
	Categories map[QuestionCategory]Metrics `json:"categories"`

	// Questions contains per-question results
	Questions []QuestionResult `json:"questions"`

	// Duration is the total evaluation time
	Duration time.Duration `json:"duration"`
}

// Baseline represents published baseline results for comparison
type Baseline struct {
	// Model is the model name
	Model string `json:"model"`

	// F1 is the overall F1 score
	F1 float64 `json:"f1"`

	// Source is where the baseline comes from
	Source string `json:"source"`
}

// PublishedBaselines contains baseline results from the LoCoMo paper
var PublishedBaselines = []Baseline{
	{Model: "Human", F1: 87.9, Source: "LoCoMo Paper"},
	{Model: "GPT-4", F1: 32.1, Source: "LoCoMo Paper"},
	{Model: "GPT-3.5", F1: 24.2, Source: "LoCoMo Paper"},
	{Model: "Llama-2-70B", F1: 16.9, Source: "LoCoMo Paper"},
	{Model: "Mistral-7B", F1: 13.9, Source: "LoCoMo Paper"},
}
