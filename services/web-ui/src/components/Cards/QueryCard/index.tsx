import { Button, Card, Flex, Subtitle } from '@tremor/react'
import { PlayCircleIcon } from '@heroicons/react/24/outline'

interface IQueryCard {
    title: string | undefined
    onClick: () => void
}

export default function QueryCard({ title, onClick }: IQueryCard) {
    return (
        <Card onClick={onClick} className="cursor-pointer">
            <Flex flexDirection="col" alignItems="start">
                <Subtitle className="line-clamp-1 text-gray-600 mb-2">
                    {title}
                </Subtitle>
                <Button variant="light" icon={PlayCircleIcon} size="sm">
                    Run query
                </Button>
            </Flex>
        </Card>
    )
}
