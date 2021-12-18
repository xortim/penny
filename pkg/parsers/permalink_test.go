package parsers

import (
	"reflect"
	"testing"

	"github.com/slack-go/slack"
)

func TestNewRefToMessageFromPermalink(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name  string
		args  args
		want  slack.ItemRef
		want1 bool
	}{
		{
			name: "Post with no replies",
			args: args{str: "https://orgname.slack.com/archives/C02BZ36790B/p1639843883000100"},
			want: slack.ItemRef{
				Channel:   "C02BZ36790B",
				Timestamp: "1639843883.000100",
			},
			want1: false,
		},
		{
			name: "Post with replies",
			args: args{str: "https://orgname.slack.com/archives/C02BZ36790B/p1639844350001200?thread_ts=1639844350.001200&amp;cid=C02BZ36790B"},
			want: slack.ItemRef{
				Channel:   "C02BZ36790B",
				Timestamp: "1639844350.001200",
			},
			want1: false,
		},
		{
			name: "Reply to post",
			args: args{str: "https://orgname.slack.com/archives/C02BZ36790B/p1639843883000800?thread_ts=1639843880.000700&amp;cid=C02BZ36790B"},
			want: slack.ItemRef{
				Channel:   "C02BZ36790B",
				Timestamp: "1639843883.000800",
			},
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := NewRefToMessageFromPermalink(tt.args.str)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRefToMessageFromPermalink() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("NewRefToMessageFromPermalink() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestPermalinkPathTS(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Prefixed with P",
			args: args{str: "p1639843883000100"},
			want: "1639843883.000100",
		},
		{
			name: "Only digits",
			args: args{str: "1639843883000100"},
			want: "1639843883.000100",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PermalinkPathTS(tt.args.str); got != tt.want {
				t.Errorf("PermalinkPathTS() = %v, want %v", got, tt.want)
			}
		})
	}
}
