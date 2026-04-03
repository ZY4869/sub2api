type QueuedTask<T> = {
  task: () => Promise<T>
  resolve: (value: T | PromiseLike<T>) => void
  reject: (reason?: unknown) => void
}

export interface AsyncTaskLimiter {
  run<T>(task: () => Promise<T>): Promise<T>
}

export function createAsyncTaskLimiter(maxConcurrency: number): AsyncTaskLimiter {
  const concurrency = Math.max(1, Math.floor(maxConcurrency))
  let activeCount = 0
  const queue: Array<QueuedTask<any>> = []

  const pumpQueue = () => {
    while (activeCount < concurrency && queue.length > 0) {
      const next = queue.shift()
      if (!next) return

      activeCount += 1
      Promise.resolve()
        .then(next.task)
        .then(next.resolve, next.reject)
        .finally(() => {
          activeCount -= 1
          pumpQueue()
        })
    }
  }

  return {
    run<T>(task: () => Promise<T>): Promise<T> {
      return new Promise<T>((resolve, reject) => {
        queue.push({
          task,
          resolve,
          reject,
        })
        pumpQueue()
      })
    },
  }
}
