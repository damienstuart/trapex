
DEST=localhost:162

# All of the trap scripts expect the environment variable DEST
export DEST

for trap_test in `ls -1 trap*.sh`; do
    echo $trap_test
    ./$trap_test
done


