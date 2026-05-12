package command

func resolveTargetArgFromFlag(targetFlag string) string {
	return targetFlag
}

func resolveToArg(toFlag string, args []string) string {
	if toFlag != "" {
		return toFlag
	}
	if len(args) > 0 {
		return args[0]
	}
	return ""
}
