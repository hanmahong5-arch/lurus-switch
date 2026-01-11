#!/bin/bash
# Lurus K3s Operations Script
# Usage: ./k3s-ops.sh <command> [args]

set -e

NAMESPACE="lurus-system"

print_help() {
    echo "Lurus K3s Operations Script"
    echo ""
    echo "Usage: $0 <command> [args]"
    echo ""
    echo "Commands:"
    echo "  status              Show cluster status (nodes, pods, services)"
    echo "  logs <deployment>   Stream logs from deployment"
    echo "  restart <deployment> Rolling restart deployment"
    echo "  scale <deployment> <n> Scale deployment to n replicas"
    echo "  backup              Backup PostgreSQL database"
    echo "  deploy              Apply Kustomize configuration"
    echo "  rollback <deployment> Rollback deployment to previous version"
    echo "  events              Show recent cluster events"
    echo "  resources           Show resource usage (CPU/Memory)"
    echo "  debug               Start debug pod"
    echo ""
    echo "Examples:"
    echo "  $0 status"
    echo "  $0 logs gateway-service"
    echo "  $0 restart gateway-service"
    echo "  $0 scale gateway-service 5"
    echo "  $0 backup"
}

case "$1" in
    status)
        echo "=== Nodes ==="
        kubectl get nodes -o wide
        echo ""
        echo "=== Pods ==="
        kubectl get pods -n $NAMESPACE -o wide
        echo ""
        echo "=== Services ==="
        kubectl get svc -n $NAMESPACE
        echo ""
        echo "=== Ingress ==="
        kubectl get ingress -n $NAMESPACE
        ;;

    logs)
        if [ -z "$2" ]; then
            echo "Usage: $0 logs <deployment-name>"
            echo "Available deployments:"
            kubectl get deployments -n $NAMESPACE -o name | sed 's/deployment.apps\//  /'
            exit 1
        fi
        kubectl logs -f deployment/$2 -n $NAMESPACE --tail=100 --all-containers
        ;;

    restart)
        if [ -z "$2" ]; then
            echo "Usage: $0 restart <deployment-name>"
            echo "Available deployments:"
            kubectl get deployments -n $NAMESPACE -o name | sed 's/deployment.apps\//  /'
            exit 1
        fi
        echo "Rolling restart $2..."
        kubectl rollout restart deployment/$2 -n $NAMESPACE
        kubectl rollout status deployment/$2 -n $NAMESPACE
        echo "Restart completed!"
        ;;

    scale)
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: $0 scale <deployment-name> <replicas>"
            exit 1
        fi
        echo "Scaling $2 to $3 replicas..."
        kubectl scale deployment/$2 --replicas=$3 -n $NAMESPACE
        kubectl get deployment/$2 -n $NAMESPACE
        ;;

    backup)
        BACKUP_FILE="backup-$(date +%Y%m%d-%H%M%S).sql"
        echo "Backing up PostgreSQL to $BACKUP_FILE..."
        kubectl exec postgres-0 -n $NAMESPACE -- pg_dump -U lurus lurus > $BACKUP_FILE
        echo "Backup completed: $BACKUP_FILE"
        ls -lh $BACKUP_FILE
        ;;

    deploy)
        echo "Applying Kustomize configuration..."
        kubectl apply -k deploy/k3s/
        echo ""
        echo "Waiting for rollout..."
        kubectl rollout status deployment/gateway-service -n $NAMESPACE --timeout=300s
        kubectl rollout status deployment/new-api -n $NAMESPACE --timeout=300s
        echo ""
        echo "Deployment completed!"
        ;;

    rollback)
        if [ -z "$2" ]; then
            echo "Usage: $0 rollback <deployment-name>"
            exit 1
        fi
        echo "Rolling back $2 to previous version..."
        kubectl rollout undo deployment/$2 -n $NAMESPACE
        kubectl rollout status deployment/$2 -n $NAMESPACE
        echo "Rollback completed!"
        ;;

    events)
        echo "=== Recent Events ==="
        kubectl get events -n $NAMESPACE --sort-by='.lastTimestamp' | tail -30
        ;;

    resources)
        echo "=== Node Resources ==="
        kubectl top nodes 2>/dev/null || echo "Metrics server not available"
        echo ""
        echo "=== Pod Resources ==="
        kubectl top pods -n $NAMESPACE 2>/dev/null || echo "Metrics server not available"
        ;;

    debug)
        echo "Starting debug pod..."
        kubectl run debug --image=busybox --rm -it -n $NAMESPACE -- sh
        ;;

    help|--help|-h)
        print_help
        ;;

    *)
        print_help
        exit 1
        ;;
esac
