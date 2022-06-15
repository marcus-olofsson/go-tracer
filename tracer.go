package tracer

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

var RE_stripFnPreamble = regexp.MustCompile(`^.*\.(.*)$`)
var RE_detectFn = regexp.MustCompile(`\$FN`)

type Options struct {
	DisableTracing bool
	CustomLogger   *zerolog.Logger

	DisableDepthValue bool
	SpacesPerIndent   int `default:"2"`

	EnterMessage string `default:"ENTER: "`
	ExitMessage  string `deafult:"EXIT: "`

	DisableNesting bool
	currentDepth   int
}

func New(opts *Options) (func(string), func(...interface{}) string) {
	var option Options
	if opts != nil {
		option = *opts
	}

	if option.DisableTracing {
		return func(string) {}, func(i ...interface{}) string { return "" }
	}

	if option.CustomLogger == nil {
		logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
		option.CustomLogger = &logger
	}

	reflectedType := reflect.TypeOf(option)
	if option.EnterMessage == "" {
		field, _ := reflectedType.FieldByName("EnterMessage")
		option.EnterMessage = field.Tag.Get("default")
	}
	if option.ExitMessage == "" {
		field, _ := reflectedType.FieldByName("ExitMessage")
		option.ExitMessage = field.Tag.Get("default")
	}

	if option.DisableNesting {
		option.SpacesPerIndent = 0
	} else if option.SpacesPerIndent == 0 {
		field, _ := reflectedType.FieldByName("SpacesPerIndent")
		option.SpacesPerIndent, _ = strconv.Atoi(field.Tag.Get("default"))
	}

	_spacify := func() string {
		spaces := strings.Repeat(" ", option.currentDepth*option.SpacesPerIndent)
		if !option.DisableDepthValue {
			return fmt.Sprintf("[%2d]%s", option.currentDepth, spaces)
		}

		return spaces
	}

	_decrementDepth := func() {
		option.currentDepth -= 1
		if option.currentDepth < 0 {
			panic("Depth is negative! Should never happen!")
		}
	}

	_incrementDepth := func() {
		option.currentDepth += 1
	}

	_enter := func(args ...interface{}) string {
		defer _incrementDepth()

		fnName := "<unknown>"
		pc, _, _, ok := runtime.Caller(1)
		if ok {
			fnName = RE_stripFnPreamble.ReplaceAllString(runtime.FuncForPC(pc).Name(), "$1")
		}

		traceMessage := fnName
		if len(args) > 0 {
			if fmtStr, ok := args[0].(string); ok {
				traceMessage = fmt.Sprintf(fmtStr, args[1:]...)
			}
		}

		traceMessage = RE_detectFn.ReplaceAllString(traceMessage, fnName)

		option.CustomLogger.Trace().Msgf("%s%s%s\n", _spacify(), option.EnterMessage, traceMessage)
		return traceMessage
	}

	_exit := func(s string) {
		_decrementDepth()
		option.CustomLogger.Trace().Msgf("%s%s%s\n", _spacify(), option.ExitMessage, s)
	}

	return _exit, _enter
}
