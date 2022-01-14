# !/bin/sh

# mme
echo "starting... mme"
osascript -e 'tell application "Terminal" 
    do script "cd ~/Documents/GitHub/volte-simulation;go run ./mme/main.go -f ./config.yml"
end tell'
sleep 3s
# eNodeB
echo "starting... eNodeB"
osascript -e 'tell application "Terminal"
    do script "cd ~/Documents/GitHub/volte-simulation;go run .//enodeb/main.go -f ./config.yml"
end tell'
sleep 3s
osascript -e 'tell application "Terminal" 
    do script "cd ~/Documents/GitHub/Algorithm-Ex/daily;go run ./main.go"
end tell'