package server

import (
	"fmt"
	"strings"

	"github.com/ycdesu/spreaddog/pkg/bbgo"
)

func collectSessionEnvVars(sessions map[string]*bbgo.ExchangeSession) (envVars map[string]string, err error) {
	envVars = make(map[string]string)

	for _, session := range sessions {
		if len(session.Key) == 0 && len(session.Secret) == 0 {
			err = fmt.Errorf("session %s key & secret is not empty", session.Name)
			return
		}

		if len(session.EnvVarPrefix) > 0 {
			envVars[session.EnvVarPrefix+"_API_KEY"] = session.Key
			envVars[session.EnvVarPrefix+"_API_SECRET"] = session.Secret
		} else if len(session.Name) > 0 {
			sn := strings.ToUpper(session.Name)
			envVars[sn+"_API_KEY"] = session.Key
			envVars[sn+"_API_SECRET"] = session.Secret
		} else {
			err = fmt.Errorf("session %s name or env var prefix is not defined", session.Name)
			return
		}

		// reset key and secret so that we won't marshal them to the config file
		session.Key = ""
		session.Secret = ""
	}

	return
}
