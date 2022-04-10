package conversations

import (
	"reflect"
	"testing"

	"github.com/slack-go/slack"
)

func TestWhoReactedWith(t *testing.T) {
	type args struct {
		msg          slack.Message
		reactionName string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Message without target reaction",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "thumbsup",
								Count: 1,
								Users: []string{"U12345"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{},
		},
		{
			name: "Message with target reaction",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "nospam",
								Count: 1,
								Users: []string{"U12345"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{"U12345"},
		},
		{
			name: "Message with target reaction and others",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "thumbsup",
								Count: 1,
								Users: []string{"U54321"},
							},
							{
								Name:  "nospam",
								Count: 1,
								Users: []string{"U12345"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{"U12345"},
		},
		{
			name: "Message with multiple target reactions and others",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "thumbsup",
								Count: 1,
								Users: []string{"U54321"},
							},
							{
								Name:  "nospam",
								Count: 2,
								Users: []string{"U12345", "U67890"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{"U12345", "U67890"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WhoReactedWith(tt.args.msg, tt.args.reactionName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WhoReactedWith() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWhoReactedWithAsMention(t *testing.T) {
	type args struct {
		msg          slack.Message
		reactionName string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Message without target reaction",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "thumbsup",
								Count: 1,
								Users: []string{"U12345"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{},
		},
		{
			name: "Message with target reaction",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "nospam",
								Count: 1,
								Users: []string{"U12345"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{"<@U12345>"},
		},
		{
			name: "Message with target reaction and others",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "thumbsup",
								Count: 1,
								Users: []string{"U54321"},
							},
							{
								Name:  "nospam",
								Count: 1,
								Users: []string{"U12345"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{"<@U12345>"},
		},
		{
			name: "Message with multiple target reactions and others",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "thumbsup",
								Count: 1,
								Users: []string{"U54321"},
							},
							{
								Name:  "nospam",
								Count: 2,
								Users: []string{"U12345", "U67890"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{"<@U12345>", "<@U67890>"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WhoReactedWithAsMention(tt.args.msg, tt.args.reactionName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WhoReactedWithAsMention() = %v, want %v", got, tt.want)
			}
		})
	}
}
