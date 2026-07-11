# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# resque.j - background jobs on Redis, wire-compatible with Resque. Schedule a
# job onto a named queue now (`enqueue`), reserve and run it from a worker later
# (`reserve`). The Redis layout is the de-facto Resque standard - queues are
# lists at `resque:queue:NAME`, the queue registry is a set at `resque:queues`,
# and a job is the JSON envelope `{"class":"WorkerName","args":[...]}` - so a job
# Jennifer enqueues can be processed by a Ruby-resque / php-resque worker and
# vice versa. Built on the `redis` module, so it needs the default `jennifer`
# binary.
#
#     import "resque.j" as resque;
#     import "redis.j" as redis;
#
#     def db as redis.Session init redis.connect(redis.Options{host: "127.0.0.1",
#         port: 6379, security: "none", user: "", password: "", db: 0});
#     resque.enqueue($db, "email", "SendWelcome", ["user@example.com", "en"]);
#
#     # a worker: reserve, then dispatch on the class string (your code)
#     def job as resque.Job init resque.reserve($db, ["high", "email"]);
#     if (len($job.class) > 0) {
#         # if ($job.class == "SendWelcome") { sendWelcome($job.args); } ...
#     }
#
# `args` is a `list of string` - it maps to Ruby-resque's positional arguments
# (`perform(a, b)`). The class string is resolved on the *worker's* side (the
# runtime that pops the job must define a job by that name); the `resque:`
# namespace must match on both ends.
use strings;
use convert;
use json;
import "./redis.j" as redis;

# The Redis key namespace (`resque` is the Resque default; both ends must agree).
def const NAMESPACE as string init "resque";

# A reserved job: its origin `queue`, the worker `class` to run, and its string
# `args`. A drained `reserve` returns an empty `Job` (`class` is "").
export def struct Job {
    queue as string,
    class as string,
    args as list of string
};

# The on-the-wire job envelope, encoded to `{"class":...,"args":[...]}`.
def struct Payload {
    class as string,
    args as list of string
};

# A failure record pushed to the failed list.
def struct Failure {
    queue as string,
    class as string,
    args as list of string,
    error as string
};

# --- keys (private) ------------------------------------------------

func queuesKey() {
    return NAMESPACE + ":queues";
}

func queueKey(queue as string) {
    return NAMESPACE + ":queue:" + $queue;
}

func failedKey() {
    return NAMESPACE + ":failed";
}

# --- envelope (private) --------------------------------------------

# encodePayload renders the Resque job envelope for `class` + `args`.
func encodePayload(class as string, args as list of string) {
    return json.encode(Payload{class: $class, args: $args});
}

# argAt reads one `args` element as a string, whatever JSON type it holds (so a
# job enqueued elsewhere with an int / bool arg still reserves cleanly).
func argAt(node as json.Value, i as int) {
    def ptr as string init "/args/" + convert.toString($i);
    def t as string init json.typeOf($node, $ptr);
    if ($t == "string") {
        return json.asString($node, $ptr);
    }
    if ($t == "int") {
        return convert.toString(json.asInt($node, $ptr));
    }
    if ($t == "float") {
        return convert.toString(json.asFloat($node, $ptr));
    }
    if ($t == "bool") {
        return convert.toString(json.asBool($node, $ptr));
    }
    return json.encode(json.get($node, $ptr));
}

# decodeJob parses a reserved envelope into a Job tagged with its origin queue.
func decodeJob(queue as string, raw as string) {
    def node as json.Value init json.decode($raw);
    def args as list of string init [];
    def n as int init json.length($node, "/args");
    def i as int init 0;
    while ($i < $n) {
        $args[] = argAt($node, $i);
        $i = $i + 1;
    }
    return Job{queue: $queue, class: json.asString($node, "/class"), args: $args};
}

# --- producer (exported) -------------------------------------------

# enqueue schedules a job: register the queue, then push the envelope onto it.
export func enqueue(session as redis.Session, queue as string, class as string,
    args as list of string) {
    redis.command($session, ["SADD", queuesKey(), $queue]);
    redis.command($session, ["RPUSH", queueKey($queue), encodePayload($class, $args)]);
}

# --- consumer (exported) -------------------------------------------

# reserve pops the next job from the first non-empty queue, checking `queues` in
# the given priority order. When every queue is drained it returns an empty
# `Job` (test with `len(job.class) == 0`).
export func reserve(session as redis.Session, queues as list of string) {
    for (def queue in $queues) {
        def r as redis.Reply init redis.command($session, ["LPOP", queueKey($queue)]);
        if ($r.kind == "string") {
            return decodeJob($queue, $r.str);
        }
    }
    return Job{queue: "", class: "", args: []};
}

# fail records a failed job on the failed list (a simplified failure entry).
export func fail(session as redis.Session, job as Job, message as string) {
    def rec as Failure init Failure{queue: $job.queue, class: $job.class,
        args: $job.args, error: $message};
    redis.command($session, ["RPUSH", failedKey(), json.encode($rec)]);
}

# --- introspection (exported) --------------------------------------

# queueLength returns the number of pending jobs on one queue.
export func queueLength(session as redis.Session, queue as string) {
    return redis.command($session, ["LLEN", queueKey($queue)]).num;
}

# queues returns the registered queue names.
export func queues(session as redis.Session) {
    def out as list of string init [];
    for (def item in redis.command($session, ["SMEMBERS", queuesKey()]).items) {
        $out[] = $item.str;
    }
    return $out;
}

# size returns the total number of pending jobs across every registered queue.
export func size(session as redis.Session) {
    def total as int init 0;
    for (def queue in queues($session)) {
        $total = $total + queueLength($session, $queue);
    }
    return $total;
}
