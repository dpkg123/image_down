name: neofetch
  
on:
   push: 
   schedule: 
     - cron: "00,30 0,3,6,9,12,15,18 1,5,11,15,21,25,28 * *" 
   workflow_dispatch: 
  
permissions: 
   contents: write 
  
jobs: 
   build: 
     runs-on: ubuntu-latest 
  
     steps: 
     # Check out repository under $GITHUB_WORKSPACE, so the job can access it 
     - run: sudo apt install -y neofetch
     - run: sudo neofetch
