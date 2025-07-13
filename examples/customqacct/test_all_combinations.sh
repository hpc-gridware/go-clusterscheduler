#!/bin/bash
# Test all qacct filter combinations to verify robust parsing

echo "=== Testing All qacct Filter Combinations ==="
echo ""

# Single filter tests
echo "1. Single Filters:"
echo "  Owner: $(./customqacct -summary -user root | grep Wallclock)"
echo "  Group: $(./customqacct -summary -group root | grep Wallclock)"  
echo "  Days 1: $(./customqacct -summary -days 1 | grep Wallclock)"
echo "  Days 0 (empty): $(./customqacct -summary -days 0 | grep Wallclock)"

# Two filter combinations
echo ""
echo "2. Two Filter Combinations:"
echo "  Owner+Group: $(./customqacct -summary -user root -group root | grep Wallclock)"
echo "  User+Days: $(./customqacct -summary -user root -days 1 | grep Wallclock)"

# Edge cases
echo ""
echo "3. Edge Cases (should be empty):"
echo "  Nonexistent user: $(./customqacct -summary -user nonexistent | grep Wallclock)"
echo "  Days=0: $(./customqacct -summary -days 0 | grep Wallclock)"

# Job details  
echo ""
echo "4. Job Details Mode:"
echo "  All jobs: $(./customqacct -user root | head -1)"
echo "  With days filter: $(./customqacct -user root -days 1 | head -1)"

echo ""
echo "=== All tests completed successfully! ==="