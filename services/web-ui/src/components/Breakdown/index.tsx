import { useAtomValue } from 'jotai'
import dayjs, { Dayjs } from 'dayjs'
import {
    Button,
    Card,
    Flex,
    Tab,
    TabGroup,
    TabList,
    Text,
    Title,
} from '@tremor/react'
import { ChevronRightIcon } from '@heroicons/react/24/outline'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useState } from 'react'
import Chart from '../Chart'
import { dateDisplay } from '../../utilities/dateDisplay'
import { searchAtom } from '../../utilities/urlstate'

interface IBreakdown {
    activeTime?: { start: Dayjs; end: Dayjs }
    chartData: (string | number | undefined | any)[]
    oldChartData?: (string | number | undefined)[]
    seeMore?: string
    isCost?: boolean
    title?: string
    loading: boolean
    colorful?: boolean
}

export default function Breakdown({
    chartData,
    oldChartData,
    loading,
    activeTime,
    seeMore,
    isCost = false,
    title,
    colorful = false,
}: IBreakdown) {
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)
    const [selectedIndex, setSelectedIndex] = useState(1)

    return (
        <Card className="pb-0 relative h-full">
            <Flex>
                <Title className="font-semibold">{title || 'Breakdown'}</Title>
                {!!activeTime && (
                    <TabGroup
                        index={selectedIndex}
                        onIndexChange={setSelectedIndex}
                        className="w-fit rounded-lg"
                    >
                        <TabList variant="solid">
                            <Tab className="pt-0.5 pb-1">
                                <Text>
                                    {dateDisplay(
                                        activeTime?.start.startOf('day')
                                    )}
                                </Text>
                            </Tab>
                            <Tab className="pt-0.5 pb-1">
                                <Text>
                                    {dateDisplay(activeTime?.end.endOf('day'))}
                                </Text>
                            </Tab>
                        </TabList>
                    </TabGroup>
                )}
            </Flex>
            <Chart
                labels={[]}
                chartData={
                    activeTime && selectedIndex === 0 ? oldChartData : chartData
                }
                chartType="doughnut"
                isCost={isCost}
                loading={loading}
                colorful={colorful}
            />
            {!!seeMore && (
                <Button
                    variant="light"
                    icon={ChevronRightIcon}
                    iconPosition="right"
                    className="absolute bottom-6 right-6"
                    onClick={() => navigate(`${seeMore}?${searchParams}`)}
                >
                    See more
                </Button>
            )}
        </Card>
    )
}
