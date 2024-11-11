import { Card, Flex, Icon, Text, Title } from '@tremor/react'
import { DocumentDuplicateIcon } from '@heroicons/react/24/outline'

interface IGoalCard {
    title: string
}

export default function GoalCard({ title }: IGoalCard) {
    return (
        <Card>
            <Flex flexDirection="col">
                <Icon
                    icon={DocumentDuplicateIcon}
                    size="lg"
                    className="p-0 mb-4"
                />
                <Text className="text-center">{title}</Text>
            </Flex>
        </Card>
    )
}
