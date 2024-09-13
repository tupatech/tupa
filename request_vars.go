package tupa

import (
	"context"
	"net/http"
	"strings"
)

// type para evitar collisões de context key
type contextKey int

// o tipo iota é usado para criar uma sequencia em tempo de compilação
// que vai ser usada para criar uma chave unica para o contexto
const varsKey contextKey = iota

// WithVars adiciona variáveis de rota para o contexto da request
func WithVars(r *http.Request, vars map[string]string) *http.Request {
	// caso tenha parâmetros de rota, vai receber um map[<key_parametro_na_assinatura>:<valor_parametro_no_client>]
	// que vem da função extractParams
	if len(vars) == 0 {
		return r
	}
	ctx := context.WithValue(r.Context(), varsKey, vars)
	return r.WithContext(ctx)
}

// Will return any route variable if exists
// vai retornar qualquer variavel de rota - caso exista
func Vars(r *http.Request) map[string]string {
	if reqVal := r.Context().Value(varsKey); reqVal != nil {
		return reqVal.(map[string]string)
	}
	return nil
}

// Extract parameters from the URL. It will filter separating by '/' and get the parameters that start with ':'
func extractParams(pattern, path string) map[string]string {
	// patternParts pega todas subrotas do endpoint, com os { } (caso exista).
	// Nesse caso será o nome completo NO ENDPOINT e não na chamada pelo client, pois 'pattern' retorna a assinatura do endpoint
	patternParts := strings.Split(pattern, "/")
	// pathParts pega o path completo sem os divisores "/" - retorna um array
	// Nesse caso, não será a assinatura e sim os reais valores do endpoint na chamada do client
	pathParts := strings.Split(path, "/")
	params := make(map[string]string)

	if len(patternParts) != len(pathParts) {
		return params
	}

	for i, patternPart := range patternParts {
		if strings.HasPrefix(patternPart, "{") && strings.HasSuffix(patternPart, "}") {
			// removendo os { }
			paramKeySignature := patternPart[1 : len(patternPart)-1]
			// se o index é válido, pega a parte correspondente e assina ao parametro
			// e.g. pattern = '/{hello} e o path = '/hey', vai extrair o 'hey' e armazenar em 'params'
			// funciona pois patternParts e pathParts vão ter a mesma quantidade de elementos, mas muda os valores - caso tenha parametro de rota
			if i < len(pathParts) {
				params[paramKeySignature] = pathParts[i]
			}
		} else if patternPart != pathParts[i] {
			// retorna um map vazio caso não tenha parâmetros
			return make(map[string]string)
		}
	}

	// caso tenha parâmetro retorna map[<key_parametro_na_assinatura>:<valor_parametro_no_client>]
	return params
}
