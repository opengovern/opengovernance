import { Badge, BadgeDelta, Card, Flex, Grid, Text, Title } from '@tremor/react'
import KPISingleItem, { IKPISingleItem } from '../KPISingleItem'

interface IProbs {
    mainTitle: string
    mainValue: number
    mainChange: number
    otherKpis: IKPISingleItem[]
}

function WhatIsDeltaType(value: number) {
    if (value > 0) return 'increase'
    if (value < 0) return 'decrease'
    return 'unchanged'
}

export default function AdvancedKPIGroup({
    mainTitle,
    mainValue,
    mainChange,
    otherKpis,
}: IProbs) {
    return (
        <Card>
            <Flex className="h-full py-1">
                <Flex
                    flexDirection="col"
                    alignItems="start"
                    justifyContent="between"
                    className="w-72 h-28 border-r border-gray-100 mr-6"
                >
                    <text className="text-lg font-bold">{mainTitle}</text>
                    <Flex className="w-fit gap-3" alignItems="end">
                        <text className="text-5xl font-bold">{mainValue}</text>
                        <Flex
                            flexDirection="col"
                            alignItems="start"
                            className="gap-1 pb-1"
                        >
                            <Text className="!text-xs">issues</Text>
                            <BadgeDelta
                                deltaType={WhatIsDeltaType(mainChange)}
                                size="xs"
                            >
                                {mainChange === 0 ? '0' : mainChange}
                                <span className="ml-0.5">%</span>
                            </BadgeDelta>
                        </Flex>
                    </Flex>
                </Flex>
                <Grid numItems={2} className="w-full gap-6">
                    {otherKpis.map((i) => (
                        <KPISingleItem
                            title={i.title}
                            value={i.value}
                            change={i.change}
                        />
                    ))}
                </Grid>
            </Flex>
        </Card>
    )
}
