# Examples

The repository's `examples/` directory holds these plus more (`showcase.j`,
`wordcount.j`) - golden-tested by `cmd/jennifer/examples_test.go`.

## Strings

```jennifer
# greeting.j
use io;

def name as string init "Jennifer";
printf("hello, " + $name + "!\n");
```

Output:

```
hello, Jennifer!
```

## FizzBuzz

```jennifer
# fizzbuzz.j
use io;

for (def i as int init 1; $i <= 15; $i = $i + 1) {
    if ($i % 15 == 0) {
        printf("FizzBuzz\n");
    } elseif ($i % 3 == 0) {
        printf("Fizz\n");
    } elseif ($i % 5 == 0) {
        printf("Buzz\n");
    } else {
        printf("%d\n", $i);
    }
}
```

## Factorial (recursion + parameters)

```jennifer
# factorial.j
use io;

func fact(n as int) {
    if ($n == 0) { return 1; }
    return $n * fact($n - 1);
}

for (def i as int init 0; $i <= 8; $i = $i + 1) {
    printf("%d! = %d\n", $i, fact($i));
}
```

## More substantive examples

For programs that exercise the full feature surface - lists, maps,
iteration, the `core` and `strings` libraries - see `examples/showcase.j`
(every feature in one file) and `examples/wordcount.j` (word-frequency
analyzer with histogram, nested aggregation, and a 3x3 grid demo). Both
are part of the golden test suite.
