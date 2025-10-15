package telegram

import (
	"testing"
)

func TestParseCommand_Status(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantCmd string
		wantErr bool
	}{
		{"simple status", "/status", "status", false},
		{"uppercase", "/STATUS", "status", false},
		{"with spaces", "/status  ", "status", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := ParseCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && args.Command != tt.wantCmd {
				t.Errorf("ParseCommand() command = %v, want %v", args.Command, tt.wantCmd)
			}
		})
	}
}

func TestParseCommand_Price(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSymbol string
		wantErr    bool
	}{
		{"with symbol", "/price BTCUSDT", "BTCUSDT", false},
		{"short symbol", "/price BTC", "BTCUSDT", false},
		{"lowercase", "/price btc", "BTCUSDT", false},
		{"no symbol", "/price", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := ParseCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && args.Symbol != tt.wantSymbol {
				t.Errorf("ParseCommand() symbol = %v, want %v", args.Symbol, tt.wantSymbol)
			}
		})
	}
}

func TestParseCommand_Buy(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSymbol string
		wantAmount float64
		wantErr    bool
	}{
		{"with amount", "/buy BTCUSDT 20", "BTCUSDT", 20.0, false},
		{"only amount", "/buy 20", "", 20.0, false},
		{"only symbol", "/buy BTC", "BTCUSDT", 0.0, false},
		{"no args", "/buy", "", 0.0, false},
		{"comma decimal", "/buy BTC 20,5", "BTCUSDT", 20.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := ParseCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if args.Symbol != tt.wantSymbol {
					t.Errorf("ParseCommand() symbol = %v, want %v", args.Symbol, tt.wantSymbol)
				}
				if args.Amount != tt.wantAmount {
					t.Errorf("ParseCommand() amount = %v, want %v", args.Amount, tt.wantAmount)
				}
			}
		})
	}
}

func TestParseCommand_Sell(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPercent float64
		wantSymbol  string
		wantErr     bool
	}{
		{"valid percent", "/sell 50", 50.0, "", false},
		{"with symbol", "/sell 50 BTCUSDT", 50.0, "BTCUSDT", false},
		{"with percent sign", "/sell 50%", 50.0, "", false},
		{"invalid percent", "/sell 0", 0.0, "", true},
		{"over 100", "/sell 150", 150.0, "", true},
		{"no args", "/sell", 0.0, "", true},
		{"comma decimal", "/sell 33,5", 33.5, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := ParseCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if args.Percent != tt.wantPercent {
					t.Errorf("ParseCommand() percent = %v, want %v", args.Percent, tt.wantPercent)
				}
				if args.Symbol != tt.wantSymbol {
					t.Errorf("ParseCommand() symbol = %v, want %v", args.Symbol, tt.wantSymbol)
				}
			}
		})
	}
}

func TestParseCommand_AutoSell(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantTrigger float64
		wantPercent float64
		wantErr     bool
	}{
		{"valid", "/autosell BTCUSDT 15 50", 15.0, 50.0, false},
		{"no symbol", "/autosell 15 50", 15.0, 50.0, false},
		{"with percent signs", "/autosell 15% 50%", 15.0, 50.0, false},
		{"invalid trigger", "/autosell 0 50", 0.0, 50.0, true},
		{"invalid percent", "/autosell 15 0", 15.0, 0.0, true},
		{"missing args", "/autosell 15", 0.0, 0.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := ParseCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if args.Trigger != tt.wantTrigger {
					t.Errorf("ParseCommand() trigger = %v, want %v", args.Trigger, tt.wantTrigger)
				}
				if args.Percent != tt.wantPercent {
					t.Errorf("ParseCommand() percent = %v, want %v", args.Percent, tt.wantPercent)
				}
			}
		})
	}
}

func TestParseCommand_GridInit(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSymbol  string
		wantLevels  int
		wantSpacing float64
		wantAmount  float64
		wantErr     bool
	}{
		{"valid", "/gridinit ETHUSDT 10 2.5 100", "ETHUSDT", 10, 2.5, 100.0, false},
		{"short symbol", "/gridinit ETH 10 2.5 100", "ETHUSDT", 10, 2.5, 100.0, false},
		{"missing args", "/gridinit ETHUSDT 10", "", 0, 0, 0, true},
		{"invalid levels", "/gridinit ETHUSDT 0 2.5 100", "", 0, 0, 0, true},
		{"invalid spacing", "/gridinit ETHUSDT 10 0 100", "", 0, 0, 0, true},
		{"comma decimal", "/gridinit ETH 10 2,5 100", "ETHUSDT", 10, 2.5, 100.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := ParseCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if args.Symbol != tt.wantSymbol {
					t.Errorf("ParseCommand() symbol = %v, want %v", args.Symbol, tt.wantSymbol)
				}
				if args.Levels != tt.wantLevels {
					t.Errorf("ParseCommand() levels = %v, want %v", args.Levels, tt.wantLevels)
				}
				if args.Spacing != tt.wantSpacing {
					t.Errorf("ParseCommand() spacing = %v, want %v", args.Spacing, tt.wantSpacing)
				}
				if args.Amount != tt.wantAmount {
					t.Errorf("ParseCommand() amount = %v, want %v", args.Amount, tt.wantAmount)
				}
			}
		})
	}
}

func TestParseCommand_History(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSymbol string
		wantCount  int
		wantErr    bool
	}{
		{"no args", "/history", "", 10, false},
		{"with count", "/history 20", "", 20, false},
		{"with symbol", "/history BTCUSDT", "BTCUSDT", 10, false},
		{"with both", "/history BTCUSDT 20", "BTCUSDT", 20, false},
		{"count first", "/history 15 BTCUSDT", "", 15, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := ParseCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if args.Symbol != tt.wantSymbol {
					t.Errorf("ParseCommand() symbol = %v, want %v", args.Symbol, tt.wantSymbol)
				}
				if args.Count != tt.wantCount {
					t.Errorf("ParseCommand() count = %v, want %v", args.Count, tt.wantCount)
				}
			}
		})
	}
}

func TestParseCommand_PanicStop(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantAction string
		wantErr    bool
	}{
		{"on", "/panicstop on", "on", false},
		{"off", "/panicstop off", "off", false},
		{"no arg", "/panicstop", "status", false},
		{"invalid arg", "/panicstop maybe", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := ParseCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && args.Action != tt.wantAction {
				t.Errorf("ParseCommand() action = %v, want %v", args.Action, tt.wantAction)
			}
		})
	}
}

func TestNormalizeSymbol(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"full symbol", "BTCUSDT", "BTCUSDT"},
		{"short symbol", "BTC", "BTCUSDT"},
		{"lowercase", "btc", "BTCUSDT"},
		{"mixed case", "BtC", "BTCUSDT"},
		{"eth", "ETH", "ETHUSDT"},
		{"with spaces", " BTC ", "BTCUSDT"},
		{"unknown", "UNKNOWN", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeSymbol(tt.input); got != tt.want {
				t.Errorf("normalizeSymbol() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{"simple", "10", 10.0},
		{"decimal", "10.5", 10.5},
		{"comma", "10,5", 10.5},
		{"with percent", "50%", 50.0},
		{"with spaces", "1 000", 1000.0},
		{"negative", "-5", -5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseFloat(tt.input); got != tt.want {
				t.Errorf("parseFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractNumbers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []float64
	}{
		{"single", "buy 20 USDT", []float64{20}},
		{"multiple", "sell 50% at 15", []float64{50, 15}},
		{"decimal", "buy 20.5 USDT", []float64{20.5}},
		{"comma", "buy 20,5 USDT", []float64{20.5}},
		{"none", "show status", []float64{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractNumbers(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractNumbers() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExtractNumbers()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractSymbols(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"single", "buy BTC", []string{"BTCUSDT"}},
		{"multiple", "buy BTC and ETH", []string{"BTCUSDT", "ETHUSDT"}},
		{"full symbol", "buy BTCUSDT", []string{"BTCUSDT"}},
		{"none", "show status", []string{}},
		{"lowercase", "buy btc", []string{"BTCUSDT"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractSymbols(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractSymbols() len = %v, want %v (got: %v)", len(got), len(tt.want), got)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExtractSymbols()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
