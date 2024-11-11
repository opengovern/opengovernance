import { Card, Flex, Text, Title } from '@tremor/react'
import DevOps from '../../../icons/persona/devops.png'
import Engineer from '../../../icons/persona/engineer.png'
import Product from '../../../icons/persona/product.png'
import Security from '../../../icons/persona/security.png'
import Executive from '../../../icons/persona/executive.png'

interface IPersonaCard {
    type: 'Engineer' | 'DevOps' | 'Product' | 'Security' | 'Executive'
}

const personaImg = (type: string) => {
    switch (type) {
        case 'Engineer':
            return Engineer
        case 'DevOps':
            return DevOps
        case 'Product':
            return Product
        case 'Security':
            return Security
        case 'Executive':
            return Executive
        default:
            return Engineer
    }
}

export default function PersonaCard({ type }: IPersonaCard) {
    return (
        <Card className="cursor-pointer">
            <Flex flexDirection="col" className="gap-3">
                <img src={personaImg(type)} alt={`${type} persona`} />
                <Text>{type}</Text>
            </Flex>
        </Card>
    )
}
