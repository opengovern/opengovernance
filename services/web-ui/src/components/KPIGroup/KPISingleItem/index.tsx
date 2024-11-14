import { ChevronRightIcon } from '@heroicons/react/20/solid'
import { Badge, Flex, Metric, Text, Title } from '@tremor/react'
import { useState } from 'react'
import LightBadge from '../../LightBadge'

export interface IKPISingleItem {
    title: string
    value: number
    change: number
}

export default function KPISingleItem({
    title,
    value,
    change,
}: IKPISingleItem) {
    const [chevronShow, setChevronShow] = useState<boolean>(false)
    return (
        <Flex
            flexDirection="col"
            alignItems="start"
            className="rounded-md cursor-pointer"
            onMouseEnter={() => setChevronShow(true)}
            onMouseLeave={() => setChevronShow(false)}
        >
            <Text className="!text-xs truncate">{title}</Text>
            <Flex justifyContent="between">
                <Flex className="w-fit gap-2" alignItems="end">
                    <Title className="font-bold">{value}</Title>
                    <LightBadge value={change} />
                </Flex>
                {chevronShow && (
                    <ChevronRightIcon className="text-gray-400 w-4" />
                )}
            </Flex>
        </Flex>
    )
}
