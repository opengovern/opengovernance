import { ChartBarIcon } from '@heroicons/react/24/outline'
import { Flex, Text } from '@tremor/react'

const easeOfSolution = (easiness: 'easy' | 'medium' | 'hard') => {
    if (easiness) {
        if (easiness === 'hard') {
            return (
                <Flex
                    flexDirection="col"
                    justifyContent="end"
                    className="gap-3 w-full h-full"
                >
                    {[1, 1, 1].map((i) => {
                        return (
                            <div className="min-w-4 min-h-4 w-full h-full rounded-md bg-red-600" />
                        )
                    })}
                </Flex>
            )
        }
        if (easiness === 'medium') {
            return (
                <Flex
                    flexDirection="col"
                    justifyContent="end"
                    className="gap-3 w-full h-full"
                >
                    <div className="min-w-4 min-h-4 w-full h-full rounded-md bg-gray-100" />
                    <div className="min-w-4 min-h-4 w-full h-full rounded-md bg-orange-500" />
                    <div className="min-w-4 min-h-4 w-full h-full rounded-md bg-orange-500" />
                </Flex>
            )
        }
        if (easiness === 'easy') {
            return (
                <Flex
                    flexDirection="col"
                    justifyContent="end"
                    className="gap-3 w-full h-full"
                >
                    <div className="min-w-4 min-h-4 w-full h-full rounded-md bg-gray-100" />
                    <div className="min-w-4 min-h-4 w-full h-full rounded-md bg-gray-100" />
                    <div className="min-w-4 min-h-4 w-full h-full rounded-md bg-yellow-500" />
                </Flex>
            )
        }
    }
    return ''
}

interface IProbs {
    isEmpty: boolean
    scalability: 'easy' | 'medium' | 'hard'
    complexity: 'easy' | 'medium' | 'hard'
    disruptivity: 'easy' | 'medium' | 'hard'
}

export default function EaseOfSolutionChart({
    isEmpty,
    scalability,
    complexity,
    disruptivity,
}: IProbs) {
    return (
        <Flex
            flexDirection="col"
            className="w-full h-full"
            alignItems="start"
            justifyContent="start"
        >
            <Text className=" font-bold mb-4 text-gray-400 ">
                Ease of Solution
            </Text>
            <Flex className="h-56 gap-3">
                <Flex flexDirection="col" className="w-fit h-full gap-3 pb-8">
                    <Flex className="h-full" justifyContent="end">
                        <Text className="text-gray-900">Hard</Text>
                    </Flex>
                    <Flex className="h-full" justifyContent="end">
                        <Text className="text-gray-900">Medium</Text>
                    </Flex>
                    <Flex className="h-full" justifyContent="end">
                        <Text className="text-gray-900">Easy</Text>
                    </Flex>
                </Flex>
                <Flex flexDirection="col" className="w-full h-full gap-2">
                    <Flex
                        justifyContent="center"
                        alignItems="center"
                        className="w-full h-full gap-3 pb-3 pl-3 border-b border-l"
                    >
                        {isEmpty ? (
                            <div className="text-center">
                                <ChartBarIcon className="mx-auto h-7 w-7 text-tremor-content-subtle dark:text-dark-tremor-content-subtle" />
                                <p className="mt-2 text-tremor-default font-medium text-tremor-content-strong dark:text-dark-tremor-content-strong">
                                    No data to show
                                </p>
                            </div>
                        ) : (
                            <>
                                {easeOfSolution(scalability)}
                                {easeOfSolution(complexity)}
                                {easeOfSolution(disruptivity)}
                            </>
                        )}
                    </Flex>
                    <Flex className="gap-3">
                        <Text className="w-full text-center text-gray-900">
                            Scalability
                        </Text>
                        <Text className="w-full text-center text-gray-900">
                            Complexity
                        </Text>
                        <Text className="w-full text-cente text-gray-900">
                            Disruptivity
                        </Text>
                    </Flex>
                </Flex>
            </Flex>
        </Flex>
    )
}
