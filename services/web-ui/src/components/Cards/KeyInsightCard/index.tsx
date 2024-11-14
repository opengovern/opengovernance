import { Card, Flex, Icon, Title } from '@tremor/react'
import {
    CloudIcon,
    FingerPrintIcon,
    KeyIcon,
    ServerStackIcon,
    ShieldExclamationIcon,
} from '@heroicons/react/24/outline'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAtomValue } from 'jotai'
import { badgeDelta } from '../../../utilities/deltaType'
import { numericDisplay } from '../../../utilities/numericDisplay'
import { searchAtom } from '../../../utilities/urlstate'

interface IKeyInsightCard {
    title: string | undefined
    prevCount: number | undefined
    count: number | undefined
    id: string | number | undefined
}

const iconGenerator = (t: string) => {
    let icon = ServerStackIcon
    if (t.includes('Issues') || t.includes('Risky')) {
        icon = ShieldExclamationIcon
    } else if (t.includes('Cloud')) {
        icon = CloudIcon
    } else if (t.includes('Disks')) {
        icon = KeyIcon
    } else if (t.includes('Logging')) {
        icon = FingerPrintIcon
    }
    return icon
}

export default function KeyInsightCard({
    title,
    prevCount,
    count,
    id,
}: IKeyInsightCard) {
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)

    return (
        <Card
            key={title}
            className="cursor-pointer"
            onClick={() => navigate(`key_insight_${id}?${searchParams}`)}
        >
            <Flex flexDirection="col" alignItems="start" className="h-full">
                <Flex flexDirection="col" alignItems="start" className="h-fit">
                    <Icon
                        icon={iconGenerator(String(title))}
                        color="blue"
                        size="lg"
                        className="p-0 mb-3"
                    />
                    <Title className="font-semibold mb-2 h-16">
                        {title}
                        <span className="ml-1 font-medium text-sm text-gray-400">
                            ({numericDisplay(count)})
                        </span>
                    </Title>
                    {badgeDelta(prevCount, count)}
                </Flex>
            </Flex>
        </Card>
    )
}
