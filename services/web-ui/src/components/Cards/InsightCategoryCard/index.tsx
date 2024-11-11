import { ChevronRightIcon } from '@heroicons/react/24/outline'
import {
    Flex,
    Button,
    ProgressCircle,
    Badge,
    Text,
    Metric,
    Icon,
} from '@tremor/react'
import { useAtomValue } from 'jotai'
import { useNavigate, useParams } from 'react-router-dom'
import { searchAtom } from '../../../utilities/urlstate'

const BoldFirstLetter = (text: string) => {
    if (!text) return null // Return null if text is empty or undefined
    const inputText = text
    const firstLetter = inputText.charAt(0) // Get the first letter
    const restOfText = inputText.slice(1) // Get the rest of the text

    return (
        <span>
            <strong>{firstLetter}</strong>
            {restOfText}
        </span>
    )
}

interface IInsightCategoryCard {
    title: string
    value: number
    category: string
    icon: any
    color: any
}

export default function InsightCategoryCard({
    title,
    value,
    category,
    icon,
    color,
}: IInsightCategoryCard) {
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)

    let insightCondition = { text: 'none', color: 'gray' }

    if (value >= 75) {
        insightCondition = { text: 'very good', color: 'emerald' }
    } else if (value >= 50 && value < 75) {
        insightCondition = { text: 'good', color: 'lime' }
    } else if (value >= 25 && value < 50) {
        insightCondition = { text: 'bad', color: 'yellow' }
    } else if (value >= 0 && value < 25) {
        insightCondition = { text: 'very bad', color: 'red' }
    }

    return (
        <Flex className="gap-8 px-8 py-6 bg-white rounded-2xl shadow-sm hover:shadow-lg hover:cursor-pointer">
            <Flex className="relative w-fit">
                <ProgressCircle color={color} value={value} size="xl" />
                <Flex
                    className={`flex flex-col justify-start pt-2 w-32 h-32 rounded-full bg-${color}-50 z-10 absolute center top-[50%] left-[50%] -translate-x-1/2 -translate-y-1/2`}
                >
                    <Icon size="lg" color={color} icon={icon} />
                    <Metric>{value}%</Metric>
                    <Text color={color}>compliant</Text>
                </Flex>
            </Flex>

            <Flex alignItems="start" flexDirection="col" className="gap-6">
                <Flex alignItems="start" flexDirection="col" className="gap-4 ">
                    <text className=" text-2xl">{BoldFirstLetter(title)}</text>
                    <Flex justifyContent="start" className="gap-2">
                        <Badge size="md" color={insightCondition.color}>
                            {insightCondition.text}
                        </Badge>
                        <Text color="gray">compliance level</Text>
                    </Flex>
                </Flex>

                <Button
                    variant="light"
                    className="hidden"
                    onClick={() =>
                        navigate(
                            `/insights?category=${category}&${searchParams}`
                        )
                    }
                >
                    {' '}
                    view insights
                </Button>
            </Flex>
            <Icon size="xl" icon={ChevronRightIcon} />
        </Flex>
    )
}
