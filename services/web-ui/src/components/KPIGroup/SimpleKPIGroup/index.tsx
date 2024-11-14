import { BadgeDelta, Card, Flex, Grid, Title } from '@tremor/react'
import KPISingleItem, { IKPISingleItem } from '../KPISingleItem'

interface IProbs {
    mainTitle: string
    otherKpis: IKPISingleItem[]
}

export default function SimpleKPIGroup({ mainTitle, otherKpis }: IProbs) {
    return (
        <Card>
            <Flex
                flexDirection="col"
                alignItems="start"
                className="w-full gap-8"
            >
                <text className="text-lg font-bold">{mainTitle}</text>

                <Flex
                    flexDirection="col"
                    alignItems="start"
                    className="w-full gap-6"
                >
                    {otherKpis.map((i) => (
                        <KPISingleItem
                            title={i.title}
                            value={i.value}
                            change={i.change}
                        />
                    ))}
                </Flex>
            </Flex>
        </Card>
    )
}
