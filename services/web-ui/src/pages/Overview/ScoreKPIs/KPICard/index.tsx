import { ChevronRightIcon } from '@heroicons/react/24/outline'
import {
    Flex,
    Text,
    Title,
    ProgressCircle,
    Button,
    Grid,
    Subtitle,
} from '@tremor/react'
import { useSetAtom } from 'jotai'
import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
interface Props {
    name: string
    number?: number
    percentage: number
    link: string
}

export default function KPICard({ name, number, percentage, link }: Props) {
    const navigate = useNavigate()
    let color = 'blue'
    if (percentage >= 75) {
        color = 'emerald'
    } else if (percentage >= 50 && percentage < 75) {
        color = 'lime'
    } else if (percentage >= 25 && percentage < 50) {
        color = 'yellow'
    } else if (percentage >= 0 && percentage < 25) {
        color = 'red'
    }

    return (
        <>
            <div
                onClick={() => {
                    navigate(link)
                }}
                key={name}
                className=" cursor-pointer flex items-center justify-between px-4 pr-0"
            >
                <div>
                    <p className="text  text-[16px] font-sans text-gray-900 dark:text-gray-50">
                        {name}
                    </p>
                    <p className="mt-1 text-[14px]  text-gray-900 dark:text-gray-50">
                        {`${name == 'Efficiency' ? '$' : ''}${Math.floor(
                            number ?? 0
                        )} `}
                        <span className="text-xs font-normal text-gray-700 dark:text-gray-300">
                            {name == 'Efficiency'
                                ? 'Waste '
                                : ' Incidents '}
                        </span>
                        {/* <span className="text-xs font-normal text-gray-700 dark:text-gray-300">
                            ({Math.floor(percentage)} &#37;)
                        </span> */}
                    </p>
                </div>
                <div className="flex flex-row gap-2 justify-center items-center">
                    <ProgressCircle
                        value={percentage}
                        radius={25}
                        strokeWidth={4}
                        // variant="neutral"
                        // className='bg-red'
                        color={color}
                        size="lg"
                    >
                        <Text>{Math.floor(percentage)}%</Text>
                    </ProgressCircle>
                    <ChevronRightIcon className="w-4 h-4" />
                </div>
            </div>
        </>
    )
}
