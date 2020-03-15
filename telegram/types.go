package telegram

import "encoding/json"

type BaseResp struct {
	Ok     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
}

type Chat struct {
	Id int64 `json:"id"`
}

type Message struct {
	Chat      *Chat   `json:"chat,omitempty"`
	MessageId int64   `json:"message_id"`
	Text      *string `json:"text,omitempty"`
}

type PollType string

const (
	QuizPollType    = "quiz"
	RegularPollType = "regular"
)

type PollOptions struct {
	Text       string `json:"text"`
	VoterCount int    `json:"voter_count"`
}

type Poll struct {
	Id                    string        `json:"id"`
	Question              string        `json:"question"`
	Options               []PollOptions `json:"options"`
	TotalVoterCount       int           `json:"total_voter_count"`
	IsClosed              bool          `json:"is_closed"`
	IsAnonymous           bool          `json:"is_anonymous"`
	Type                  PollType      `json:"type"`
	AllowsMultipleAnswers bool          `json:"allows_multiple_answers"`
	CorrectOptionId       *int          `json:"correct_option_id,omitempty"`
}

type Update struct {
	UpdateId int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
	Poll     *Poll    `json:"poll,omitempty"`
}
