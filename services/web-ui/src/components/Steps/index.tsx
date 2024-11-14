import { Flex } from '@tremor/react'
import { CheckIcon } from '@heroicons/react/20/solid'

interface ISteps {
    steps: number
    currentStep: number
}

const stepNavigation = (status: string) => {
    switch (status) {
        case 'complete':
            return (
                <>
                    <Flex className="absolute inset-0" aria-hidden="true">
                        <div className="h-0.5 w-full bg-openg-600" />
                    </Flex>
                    <Flex
                        alignItems="center"
                        justifyContent="center"
                        className="relative h-8 w-8 rounded-full bg-openg-600"
                    >
                        <CheckIcon
                            className="h-5 w-5 text-white"
                            aria-hidden="true"
                        />
                    </Flex>
                </>
            )
        case 'current':
            return (
                <>
                    <Flex className="absolute inset-0" aria-hidden="true">
                        <div className="h-0.5 w-full bg-gray-200" />
                    </Flex>
                    <Flex
                        alignItems="center"
                        justifyContent="center"
                        className="relative h-8 w-8 rounded-full border-2 border-openg-600 bg-white"
                        aria-current="step"
                    >
                        <span
                            className="h-2.5 w-2.5 rounded-full bg-openg-600"
                            aria-hidden="true"
                        />
                    </Flex>
                </>
            )
        default:
            return (
                <>
                    <Flex className="absolute inset-0" aria-hidden="true">
                        <div className="h-0.5 w-full bg-gray-200" />
                    </Flex>
                    <Flex
                        alignItems="center"
                        justifyContent="center"
                        className="group relative h-8 w-8 rounded-full border-2 border-gray-300 bg-white"
                    >
                        <span
                            className="h-2.5 w-2.5 rounded-full bg-gray-300"
                            aria-hidden="true"
                        />
                    </Flex>
                </>
            )
    }
}

const getStatus = (current: number, value: number) => {
    switch (true) {
        case current === value:
            return 'current'
        case current < value:
            return 'upcoming'
        default:
            return 'complete'
    }
}

const stepFix = (step: number) => {
    const stepss = []
    for (let i = 0; i < step; i += 1) {
        stepss.push({ id: i })
    }
    return stepss
}

export default function Steps({ steps, currentStep }: ISteps) {
    return (
        <nav className="w-full mb-6">
            <ol className="flex items-center justify-between">
                {stepFix(steps).map((step: any, stepIdx: number) => (
                    <li
                        key={step.id}
                        className={`${
                            stepIdx !== steps - 1 ? 'w-full' : ''
                        } relative`}
                    >
                        {stepNavigation(getStatus(currentStep, step.id + 1))}
                    </li>
                ))}
            </ol>
        </nav>
    )
}
