import { Flex, Tab, TabGroup, TabList, Card, Text, Icon } from '@tremor/react'
import { useState } from 'react'

type FilterTab = {
    type: number
    icon: React.ForwardRefExoticComponent<
        Omit<React.SVGProps<SVGSVGElement>, 'ref'> & {
            title?: string | undefined
            titleId?: string | undefined
        } & React.RefAttributes<SVGSVGElement>
    >
    name: string
    function: () => void
}[]

interface IProps {
    tabs: FilterTab
    tab: number
    setTab: (v: number) => void
}

export default function FilterTabs({ tabs, tab, setTab }: IProps) {
    return (
        <TabGroup index={tab} onIndexChange={setTab}>
            <TabList className="border-0">
                {tabs.map((t) => (
                    <Tab
                        key={t.name}
                        className="p-0 w-1/4 rounded-lg compliance-fix border-2 !border-b-2 hover:border-gray-200"
                        id={t.name}
                        onClick={t.function}
                    >
                        <Card className="px-4 py-3">
                            <Flex>
                                <Flex className="w-fit gap-2">
                                    <Icon
                                        icon={t.icon}
                                        className="w-6 max-w-[24px] text-openg-500"
                                    />
                                    <Text className="text-gray-800 w-fit">
                                        {t.name}
                                    </Text>
                                </Flex>
                                {/* {tab === t.type ? (
                                    <div />
                                ) : (
                                    <ChevronDownIcon className="w-4 text-openg-500" />
                                )} */}
                            </Flex>
                        </Card>
                    </Tab>
                ))}
            </TabList>
        </TabGroup>
    )
}
