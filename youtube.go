package main

import "regexp"

func getYoutubeUrl(fileToSync string) string {
	// re := regexp.MustCompile(`\r?\n`)

	// don't forget this matches a reversed youtube id
	// e.g.:
	// 4pm.]QXqBuJpErb6[ SGT - sgnihT eciN evaH tnaC eW yhW sI sihT _ SMLAER ELTTAB/sevitcepsorteR - sgniht ecin evah tnac ew yhw si sihT/tiucsiblatot
	re := regexp.MustCompile(`^[A-z0-9]{2,6}\.\]*([A-z0-9-]{11})(\[|-){1}`)
	regexSubmatches := re.FindStringSubmatch(reverse(fileToSync))

	if len(regexSubmatches) < 3 {
		// debug("regexSubmatches < 3 for %#v. returning empty string. %#v.", fileToSync, regexSubmatches)
		return ""
	}
	// debug("regexSubmatches %#v", regexSubmatches)
	youtubeId := reverse(regexSubmatches[1])
	debug("youtubeId: `%v` (%#v)", youtubeId, fileToSync)
	downloadUrl := ""
	if len(youtubeId) > 0 {
		downloadUrl = "https://youtu.be/" + youtubeId
	}

	return downloadUrl
}
