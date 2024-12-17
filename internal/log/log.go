package log

import (
	goflag "flag"

	"github.com/spf13/pflag"
	klog "k8s.io/klog/v2"
)

// AddFlags adds flags for logging.
func AddFlags(flags *pflag.FlagSet) {
	klogFlagSet := goflag.NewFlagSet("klog", goflag.ContinueOnError)
	klog.InitFlags(klogFlagSet)
	flags.AddGoFlagSet(klogFlagSet)

	// Logs are written to standard error instead of to files
	_ = flags.Set("logtostderr", "true")

	// Hide log flags to make our help command consistent with kubectl
	_ = flags.MarkHidden("add_dir_header")
	_ = flags.MarkHidden("alsologtostderr")
	_ = flags.MarkHidden("log_backtrace_at")
	_ = flags.MarkHidden("log_dir")
	_ = flags.MarkHidden("log_file")
	_ = flags.MarkHidden("log_file_max_size")
	_ = flags.MarkHidden("logtostderr")
	_ = flags.MarkHidden("one_output")
	_ = flags.MarkHidden("skip_headers")
	_ = flags.MarkHidden("skip_log_headers")
	_ = flags.MarkHidden("stderrthreshold")
	_ = flags.MarkHidden("v")
	_ = flags.MarkHidden("vmodule")
}
