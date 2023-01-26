# functions.sh

_fn_dir=$(dirname "${BASH_SOURCE[0]}")
_fn_root_dir="$_fn_dir/../../"

function ls_node_pools() {
    kubectl get nodes -L pool --no-headers | awk '{ print $6"\t"$1 }' | sort
}

alias ls_bidder="kubectl get pods -l app=bidder"
alias ls_load_generator="kubectl get pods -l app=load-generator"
alias ls_aerospike="kubectl get pods -l app=aerospike"
alias ls_datagen="kubectl get pods -l app=datagen"

alias pf_grafana="kubectl port-forward svc/prom-grafana 8080:80"
alias pf_aerospike="kubectl port-forward svc/aerospike-aerospike 3000:3000"

## Test if all pods in the bidder deployment are available and updated
function is_bidder_deployment_ready() {
    local target_replicas
    local available_replicas
    local updated_replicas
    target_replicas=$(kubectl get deployments.apps bidder -o json | jq '.spec.replicas // 0')
    available_replicas=$(kubectl get deployments.apps bidder -o json | jq '.status.availableReplicas // 0')
    updated_replicas=$(kubectl get deployments.apps bidder -o json | jq '.status.updatedReplicas // 0')
    if [[ "$available_replicas" -ne "$target_replicas" || $updated_replicas -ne "$target_replicas" ]]; then
        return 1
    fi
    return 0
}

## Test if all pods in the aerospike stateful set are ready
function is_aerospike_stateful_set_ready() {
    local target_replicas
    local ready_replicas
    target_replicas=$(kubectl get statefulsets.apps aerospike-aerospike -o json | jq '.spec.replicas // 0')
    ready_replicas=$(kubectl get statefulsets.apps aerospike-aerospike -o json | jq '.status.readyReplicas // 0')

    if [[ $ready_replicas -ne $target_replicas ]]; then
        return 1
    fi
    return 0
}

## Wait until all replicas in the aerospike stateful set are ready
function wait_for_aerospike_stateful_set() {
    while ! is_aerospike_stateful_set_ready; do
        sleep 30
    done
}

## Enforce all pods in a deployment to restart
## scaling in to 0 replicas, then back to original value,
## or value passed with `--target N` parameter.
function restart_bidder_deployment() {
    local target_replicas=-1
    local scale_out_step=-1

    while [[ $# -gt 0 ]]; do
        case $1 in
        --step)
            scale_out_step=$2
            shift 2
            ;;
        --target)
            target_replicas=$2
            shift 2
            ;;
        *)
            shift
            ;;
        esac
    done

    if [[ $target_replicas -eq -1 ]]; then
        target_replicas=$(kubectl get deployments.apps bidder -o json | jq '.spec.replicas // 0')
    fi

    if [[ $scale_out_step -eq -1 ]]; then
        scale_out_step=$target_replicas
    fi

    created=0
    for i in $(seq 0 "$scale_out_step" "$target_replicas"); do
        kubectl scale deployment bidder --replicas="$i"
        while ! is_bidder_deployment_ready; do
            sleep 10
        done
        created=$i
    done

    if [[ $created -lt $target_replicas ]]; then
        kubectl scale deployment bidder --replicas="$target_replicas"
        while ! is_bidder_deployment_ready; do
            sleep 10
        done
    fi
}

function run_bench() {
    if [[ -z $1 ]]; then
        duration=1m
    else
        duration=$1
    fi

    if [[ -z $2 ]]; then
        rate=1000
    else
        rate=$2
    fi

    if [[ -z $3 ]]; then
        replicas=1
    else
        replicas=$3
    fi

    ENABLE_PROFILER=
    if [[ -n $4 && $4 ]]; then
        echo ":::: enable profiling"
        ENABLE_PROFILER=1
    fi

    # delete the previous job; it deserves better handling
    kubectl delete jobs.batch load-generator || true

    # wait until all replicas are removed
    while (($(ls_load_generators | wc -l) > 0)); do
        sleep 1
    done

    # run benchmark
    ENABLE_PROFILER=$ENABLE_PROFILER DURATION="$duration" RATE_PER_JOB="$rate" NUMBER_OF_JOBS="$replicas" make benchmark@run

    echo "starting load generators..."
    # wait for all load-generator to start
    while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase=Running" | wc -l) > replicas)); do
        sleep 1
    done

    # wait for all load-generator to succeed
    while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase!=Succeeded" | wc -l) > 0)); do
        sleep 1
    done
}

function collect_stuff() {
    local output_dir=${1:-$PWD}
    mkdir -p "$output_dir"

    {
        # Docusaurus special formatting
        echo "---"
        echo "sidebar: false"
        echo "---"
        echo "\`\`\`"

        for pod in $(kubectl get pods -l app=load-generator -o name); do
            kubectl logs "$pod"
            printf "\n\n"
        done

        # Docusaurus special formatting
        echo "\`\`\`"
    } >"$output_dir/load_generators_logs.md"

    make -C "$_fn_root_dir" benchmark@report REPORT_FILE="$output_dir/load_generators_report.md"

    {
        # Docusaurus special formatting
        echo "---"
        echo "sidebar: false"
        echo "---"
        echo "\`\`\`"

        kubectl describe configmaps bidder-config
        echo
        echo
        kubectl get deployments.apps bidder
        echo
        echo
        kubectl get pods -l app=bidder -o wide
        echo
        echo
        kubectl describe "$(kubectl get pods -l app=bidder -o name | head -1)"
        echo
        echo
        kubectl describe jobs.batch load-generator
        echo
        echo
        kubectl get pods -l app=load-generator -o wide
        echo
        echo
        kubectl describe "$(kubectl get pods -l app=load-generator -o name | head -1)"
        echo
        echo
        kubectl get nodes -L pool

        # Docusaurus special formatting
        echo "\`\`\`"
    } >>"$output_dir/environment_details.md"
}

function abort_bench() {
    kubectl delete jobs load-generator
    kubectl scale deployment bidder --replicas=0
}

function template_with_envs() {
    if [[ $# -lt 2 ]]; then
        echo "substitute: missing arguments"
        echo "  usage: substitute template resolved [variable=value [variableN=valueN]]"
        exit 0
    fi

    template="${1}"
    shift
    resolved="${1}"
    shift
    variables=${*}

    cmd="${variables[*]} envsubst < $template > $resolved"

    eval "$cmd"
}

function _log() {
    datetime=$(date -Iseconds)
    caller=${FUNCNAME[1]}
    echo ":: $datetime :: $caller :: ${*}"
}

function wait_for_load_generators_to_complete() {
    if [[ $# -lt 1 ]]; then
        exit 1
    fi

    timeout=${1}

    _log "waiting for load generators to complete"
    while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase=Succeeded" | wc -l) == 0)); do
        sleep 1
    done
    _log "first load generator completed, waiting ${timeout} seconds for the other"
    t=0
    while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase!=Succeeded" | wc -l) > 0)); do
        sleep 1
        t=$((t + 1))
        if ((t == timeout)); then
            _log "load generators timeout to complete reached, deleting remaining"
            for pod in $(kubectl get pods -l app=load-generator -o name --field-selector "status.phase!=Succeeded"); do
                kubectl delete --force "$pod" || true
            done
            return
        fi
    done
    _log "load generators completed"
    return
}

function collect_grafana() {
    if [[ $# -ne 3 ]]; then
        echo "collect_grafana: missing arguments" >&2
        echo "  usage: collect_grafana output_file start_timestamp end_timestamp" >&2
        exit 1
    fi

    curl -f -o "$1" --user admin:prom-operator "http://localhost:8080/render/d/9BDpvv-Mz/?orgId=1&from=$2&to=$3&width=2500&height=2500&tz=Europe%2FWarsaw&kiosk=tv"
}

function collect_grafana_node() {
    if [[ $# -ne 3 ]]; then
        echo "collect_grafana: missing arguments" >&2
        echo "  usage: collect_grafana output_file start_timestamp end_timestamp" >&2
        exit 1
    fi

    local instance=$(kubectl get nodes -l "pool=application" -o jsonpath="{.items[0].metadata.name}")

    curl -f -o "$1" --user admin:prom-operator "http://localhost:8080/render/d/fa49a4706d07a042595b664c87fb33ea/nodes?orgId=1&from=$2&to=$3&width=2500&height=1200&tz=Europe%2FWarsaw&kiosk=tv&var-instance=${instance}"
}

function collect_application_logs() {
    local out="${1:-$PWD}/application_logs"

    for pod in $(kubectl get pods -l app=bidder -o name | cut -d "/" -f 2); do
        path=$(realpath "${out}/${pod}.log")
        mkdir -p "$(dirname "$path")"
        kubectl logs "$pod" >"$path"
    done
}

function collect_application_logs_stats() {
    local out=${1:-"application_logs_stats.txt"}
    mkdir -p "$(dirname "$out")"
    local tmp
    tmp=$(mktemp)
    for pod in $(kubectl get pods -l app=bidder -o name); do
        kubectl logs "$pod" | sed -E 's/[0-9]+/#/g' | sort | uniq -c >>"$tmp"
    done
    awk '{cnt=$1; $1=""; gsub(/^[ \t]+/, "", $0); arr[$0]+=cnt} END {for (i in arr) {print arr[i],i}}' "$tmp" | sort -rnk 1 >"$out"
}
