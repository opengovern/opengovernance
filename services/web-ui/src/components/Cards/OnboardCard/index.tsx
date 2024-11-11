import { Card, CategoryBar, Flex, Metric, Text } from '@tremor/react'
import Spinner from '../../Spinner'
import { numericDisplay } from '../../../utilities/numericDisplay'

interface IOnBoardCard {
    title: string
    active: number | undefined
    inProgress: number | undefined
    healthy: number | undefined
    unhealthy: number | undefined
    loading: boolean
}

export default function OnboardCard({
    title,
    active,
    healthy,
    unhealthy,
    inProgress,
    loading,
}: IOnBoardCard) {
    const total = (healthy || 0) + (unhealthy || 0) + (inProgress || 0)
    return (
        <Card className="overflow-hidden">
            <Flex flexDirection="col" alignItems="start" className="w-fit">
                <Text className="mb-1.5 whitespace-nowrap">{title}</Text>
                {loading ? (
                    <div className="w-fit">
                        <Spinner />
                    </div>
                ) : (
                    <Metric>{numericDisplay(active || 0)}</Metric>
                )}
            </Flex>
            <CategoryBar
                className="w-full mt-4 mb-2"
                values={[
                    ((inProgress || 0) / (total || 1)) * 100,
                    ((healthy || 0) / (total || 1)) * 100,
                    ((unhealthy || 0) / (total || 1)) * 100,
                ]}
                markerValue={
                    (((inProgress || 0) + (healthy || 0)) / (total || 1)) * 100
                }
                showLabels={false}
                colors={['slate', 'emerald', 'amber']}
            />
            <Flex justifyContent="start" className="gap-3">
                <Flex alignItems="start" className="gap-2 w-fit">
                    <div
                        className="mt-1.5 h-2.5 w-2.5 rounded-full"
                        style={{ backgroundColor: '#64748b' }}
                    />
                    <Text>{`In progress (${inProgress || 0})`}</Text>
                </Flex>
                <Flex alignItems="start" className="gap-2 w-fit">
                    <div
                        className="mt-1.5 h-2.5 w-2.5 rounded-full"
                        style={{ backgroundColor: '#10b981' }}
                    />
                    <Text>{`Healthy (${healthy || 0})`}</Text>
                </Flex>
                <Flex alignItems="start" className="gap-2 w-fit">
                    <div
                        className="mt-1.5 h-2.5 w-2.5 rounded-full"
                        style={{ backgroundColor: '#f59e0b' }}
                    />
                    <Text>{`Unhealthy (${unhealthy || 0})`}</Text>
                </Flex>
            </Flex>
        </Card>
    )
}
