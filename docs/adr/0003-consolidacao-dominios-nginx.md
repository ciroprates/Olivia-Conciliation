# Consolidação dos domínios no nginx

A arquitetura original expunha três domínios distintos: `console.olivinha.site` (frontend), `bff.olivinha.site` (backend API) e `api.olivinha.site` (Olivia API / execuções). Essa separação criava complexidade desnecessária de CORS e dificultava o modelo de autenticação baseado em cookies HttpOnly com CSRF double-submit, que exige same-origin.

A decisão adotada foi consolidar tudo em `console.olivinha.site`, roteando `/api/` para o backend Go e `/executions/` para a Olivia API diretamente no nginx. Os domínios anteriores foram mantidos temporariamente como redirects 308 para compatibilidade retroativa e depois removidos.

## Domínios ativos

| Domínio | Serviço |
|---|---|
| `console.olivinha.online` | Frontend + backend API (`/api/`) + execuções Pluggy (`/executions/`) |
| `n8n.olivinha.online` | Painel do n8n (workflow automation) |
| `waha.olivinha.online` | WAHA (WhatsApp API bridge) |

Todos os domínios compartilham o mesmo certificado Let's Encrypt (`olivinha.online`), gerenciado por `scripts/setup-ssl.sh`. O domínio anterior `olivinha.site` foi abandonado e substituído por corte imediato em 2026-06-19.

## Alternativa considerada

Manter domínios separados com CORS configurado explicitamente. Descartada por aumentar a superfície de configuração e não trazer benefício real dado que frontend e backend são deployados juntos no mesmo host.
