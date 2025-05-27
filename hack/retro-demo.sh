#!/bin/bash

# DOS-LLVM Retro Demonstration Script
# Showcases the MS-DOS inspired terminal interface

# Colors for retro terminal effect
GREEN='\033[0;32m'
BRIGHT_GREEN='\033[1;32m'
NC='\033[0m' # No Color

echo -e "${BRIGHT_GREEN}"
cat << "EOF"
████████╗███████╗██████╗ ███╗   ███╗██╗███╗   ██╗ █████╗ ██╗     
╚══██╔══╝██╔════╝██╔══██╗████╗ ████║██║████╗  ██║██╔══██╗██║     
   ██║   █████╗  ██████╔╝██╔████╔██║██║██╔██╗ ██║███████║██║     
   ██║   ██╔══╝  ██╔══██╗██║╚██╔╝██║██║██║╚██╗██║██╔══██║██║     
   ██║   ███████╗██║  ██║██║ ╚═╝ ██║██║██║ ╚████║██║  ██║███████╗
   ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝╚══════╝
                                                                  
   ██████╗  ██████╗ ███████╗      ██╗     ██╗     ██╗   ██╗███╗   ███╗
   ██╔══██╗██╔═══██╗██╔════╝      ██║     ██║     ██║   ██║████╗ ████║
   ██║  ██║██║   ██║███████╗█████╗██║     ██║     ██║   ██║██╔████╔██║
   ██║  ██║██║   ██║╚════██║╚════╝██║     ██║     ╚██╗ ██╔╝██║╚██╔╝██║
   ██████╔╝╚██████╔╝███████║      ███████╗███████╗ ╚████╔╝ ██║ ╚═╝ ██║
   ╚═════╝  ╚═════╝ ╚══════╝      ╚══════╝╚══════╝  ╚═══╝  ╚═╝     ╚═╝
                                                                       
              DISTRIBUTED LARGE LANGUAGE MODEL CLUSTER v1.0
EOF
echo -e "${NC}"

echo -e "${GREEN}▓▓▓ INITIALIZING RETRO TERMINAL INTERFACE ▓▓▓${NC}"
echo ""
echo -e "${GREEN}SYSTEM STATUS:${NC}"
echo -e "${GREEN}[████████████████████████████████████████] 100%${NC}"
echo ""
echo -e "${GREEN}▓▓▓ AVAILABLE COMMANDS ▓▓▓${NC}"
echo ""
echo -e "${GREEN}[TUI]     - Launch Terminal User Interface${NC}"
echo -e "${GREEN}[AGENT]   - Start Cluster Agent Node${NC}"
echo -e "${GREEN}[K8S]     - Deploy to Kubernetes${NC}"
echo -e "${GREEN}[HELP]    - Show command reference${NC}"
echo ""

# Function to display retro-style help
show_help() {
    echo -e "${BRIGHT_GREEN}"
    cat << "EOF"
┌─────────────────────────────────────────────────────────────────┐
│                        DOS-LLVM HELP SYSTEM                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  COMMAND REFERENCE:                                            │
│                                                                 │
│  ./bin/tui                   - Launch the TUI client          │
│  ./bin/agent                 - Start a cluster agent          │
│  make build                  - Build all components           │
│  make k8s-deploy             - Deploy to Kubernetes           │
│  make clean                  - Clean build artifacts          │
│                                                                 │
│  TUI NAVIGATION:                                               │
│  TAB                         - Switch between tabs            │
│  ↑/↓ or J/K                  - Navigate lists                 │
│  Q                           - Quit application               │
│                                                                 │
│  FEATURES:                                                     │
│  • Retro MS-DOS aesthetic with green terminal theme           │
│  • Real-time cluster monitoring                               │
│  • gRPC-based P2P communication                               │
│  • Kubernetes deployment support                              │
│  • Distributed LLM layer processing                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
EOF
    echo -e "${NC}"
}

# Interactive menu
while true; do
    echo -e "${GREEN}DOS-LLVM> ${NC}" 
    read -r cmd
    
    case "$cmd" in
        "TUI"|"tui")
            echo -e "${GREEN}▓▓▓ LAUNCHING TUI CLIENT ▓▓▓${NC}"
            ./bin/tui
            ;;
        "AGENT"|"agent")
            echo -e "${GREEN}▓▓▓ STARTING CLUSTER AGENT ▓▓▓${NC}"
            ./bin/agent
            ;;
        "K8S"|"k8s")
            echo -e "${GREEN}▓▓▓ DEPLOYING TO KUBERNETES ▓▓▓${NC}"
            make k8s-deploy
            ;;
        "HELP"|"help")
            show_help
            ;;
        "EXIT"|"exit"|"quit"|"q")
            echo -e "${GREEN}▓▓▓ TERMINATING DOS-LLVM SESSION ▓▓▓${NC}"
            echo -e "${GREEN}THANK YOU FOR USING DOS-LLVM v1.0${NC}"
            break
            ;;
        *)
            echo -e "${GREEN}UNKNOWN COMMAND: $cmd${NC}"
            echo -e "${GREEN}TYPE 'HELP' FOR COMMAND REFERENCE${NC}"
            ;;
    esac
    echo ""
done
