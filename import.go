package virtualbox

import "strconv"

//ImportOVF imports ova or ovf from the given path
func ImportOVF(path string, vsys int, name string) error {
	return Manage().run(
		"import", path,
		"--vsys", strconv.Itoa(vsys),
		"--vmname", name,
	)
}
