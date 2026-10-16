#!/usr/bin/env bash

set -euo pipefail

readonly host=${DEPLOY_HOST:-pi}
readonly image=ghcr.io/itztas/whatsapp-a-bot
readonly container=whatsapp-a-bot
readonly volume=whatsapp-a-bot-data
readonly env_file=/etc/whatsapp-a-bot.env
readonly login_timeout=${DEPLOY_LOGIN_TIMEOUT:-60}

die() {
    printf 'deploy: %s\n' "$1" >&2
    exit 1
}

step() {
    printf '==> %s\n' "$1"
}

ensure_no_local_bot() {
    local repo
    repo=$(git rev-parse --show-toplevel)
    if pgrep -f "^$repo/tmp/main" >/dev/null || pgrep -x whatsapp-a-bot >/dev/null; then
        die "o bot está rodando localmente; pare-o antes, senão ele toma a sessão do servidor"
    fi
}

wait_for_build() {
    local sha=$1 run_id
    run_id=$(gh run list --workflow build.yml --commit "$sha" --limit 1 --json databaseId --jq '.[0].databaseId // empty')
    [[ -n $run_id ]] || die "nenhum build do CI para ${sha:0:7}; dê push na main primeiro"
    step "aguardando o build $run_id do CI (${sha:0:7})"
    gh run watch "$run_id" --exit-status --interval 15 >/dev/null ||
        die "o build $run_id do CI não passou: gh run view $run_id --log-failed"
}

replace_container() {
    local ref=$1
    step "baixando $ref no $host"
    ssudo -H "$host" docker pull -q "$ref" >/dev/null
    step "recriando o container $container"
    ssudo -H "$host" -s "docker rm -f $container >/dev/null 2>&1 || true
docker run -d --name $container --restart unless-stopped --env-file $env_file -v $volume:/data $ref >/dev/null
docker images $image --format '{{.Repository}}:{{.Tag}}' | grep -vx '$ref' | xargs -r docker rmi >/dev/null 2>&1 || true
docker image prune -f >/dev/null"
}

wait_for_login() {
    local deadline=$((SECONDS + login_timeout)) logs
    step "aguardando o login no WhatsApp (até ${login_timeout}s)"
    while ((SECONDS < deadline)); do
        sleep 5
        logs=$(ssudo -H "$host" docker logs "$container" 2>&1)
        if [[ $logs == *"Successfully authenticated"* ]]; then
            return 0
        fi
    done
    printf '%s\n' "$logs" | tail -n 20 >&2
    die "sem login em ${login_timeout}s; se a sessão caiu, escaneie o qr code em: ssudo -H $host docker logs -f $container"
}

main() {
    local sha ref
    git fetch -q origin main
    sha=$(git rev-parse origin/main)
    ref="$image:sha-${sha:0:7}"

    ensure_no_local_bot
    wait_for_build "$sha"
    replace_container "$ref"
    wait_for_login
    step "deploy concluído: $ref rodando no $host"
}

main "$@"
