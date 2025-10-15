package iamlivecore

import "log"

// debugf prints formatted debug logs when the --debug flag is enabled.
func debugf(format string, v ...interface{}) {
	if debugFlag != nil && *debugFlag {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// debugln prints line-based debug logs when the --debug flag is enabled.
func debugln(v ...interface{}) {
	if debugFlag != nil && *debugFlag {
		log.Println(append([]interface{}{"[DEBUG]"}, v...)...)
	}
}
