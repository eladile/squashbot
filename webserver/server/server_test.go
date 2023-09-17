package server

import (
	"testing"
)

func TestServerGetTime(t *testing.T) {
	s := Server{}
	tests := []struct {
		name     string
		text     string
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		{
			name:     "basic time only",
			text:     "10:43",
			wantHour: 10,
			wantMin:  43,
			wantErr:  false,
		},
		{
			name:     "date time",
			text:     "18/09/23 10:42",
			wantHour: 10,
			wantMin:  42,
			wantErr:  false,
		},
		{
			name:    "bad time",
			text:    "18/09/23 10:2 oh yeah",
			wantErr: true,
		},
		{
			name:    "no time",
			text:    "18/09/23 noon",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHour, gotMin, err := s.getTime(tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("getTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotHour != tt.wantHour {
				t.Errorf("getTime() gotHour = %v, want %v", gotHour, tt.wantHour)
			}
			if gotMin != tt.wantMin {
				t.Errorf("getTime() gotMin = %v, want %v", gotMin, tt.wantMin)
			}
		})
	}
}

func TestServerGetDate(t *testing.T) {
	s := Server{}
	tests := []struct {
		name      string
		text      string
		wantDay   int
		wantMonth int
		wantYear  int
		wantErr   bool
	}{
		{
			name:      "basic date only",
			text:      "07/10/23",
			wantDay:   07,
			wantMonth: 10,
			wantYear:  23,
			wantErr:   false,
		},
		{
			name:      "date time",
			text:      "07/10/23 10:42",
			wantDay:   07,
			wantMonth: 10,
			wantYear:  23,
			wantErr:   false,
		},
		{
			name:      "bad date",
			text:      "18/09/2 10:22",
			wantErr:   false,
			wantDay:   18,
			wantMonth: 9,
			wantYear:  2,
		},
		{
			name:    "no date",
			text:    "noon 12:00",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDay, gotMonth, gotYear, err := s.getDate(tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("getDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotDay != tt.wantDay {
				t.Errorf("getDate() gotDay = %v, want %v", gotDay, tt.wantDay)
			}
			if gotMonth != tt.wantMonth {
				t.Errorf("getDate() gotMonth = %v, want %v", gotMonth, tt.wantMonth)
			}
			if gotYear != tt.wantYear {
				t.Errorf("getDate() gotYear = %v, want %v", gotYear, tt.wantYear)
			}
		})
	}
}
