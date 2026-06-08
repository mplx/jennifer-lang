# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# fizzbuzz.j - classic FizzBuzz from 1 to 15.
# Demonstrates M2 features: for loop, if/elseif/else, comparison, modulo.
use io;

for (def i as int init 1; $i <= 15; $i = $i + 1) {
    if ($i % 15 == 0) {
        printf("FizzBuzz\n");
    } elseif ($i % 3 == 0) {
        printf("Fizz\n");
    } elseif ($i % 5 == 0) {
        printf("Buzz\n");
    } else {
        printf($i);
        printf("\n");
    }
}
