package gwr

import "io"

// writes buff to one or more writers, returns a list of the writers that
// worked
func writeToEach(buf []byte, writers []io.Writer) []io.Writer {
	failed := make([]int, len(writers))
	for i, w := range writers {
		if _, err := w.Write(buf); err != nil {
			failed = append(failed, i)
		}
	}

	if len(failed) != 0 {
		var okay []io.Writer
		for i, w := range writers {
			if i != failed[0] {
				okay = append(okay, w)
			}
			if i >= failed[0] {
				failed = failed[1:]
				if len(failed) == 0 {
					if i < len(writers)-1 {
						okay = append(okay, writers[i+1:]...)
					}
					break
				}
			}
		}
		writers = okay
	}

	return writers
}

// TODO: a parallel alternative, say writeToMany?
