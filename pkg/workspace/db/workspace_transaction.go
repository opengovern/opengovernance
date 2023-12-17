package db

func (s *Database) CreateWorkspaceTransaction(m *WorkspaceTransaction) error {
	return s.Orm.Model(&WorkspaceTransaction{}).Create(m).Error
}

func (s *Database) GetTransactionsByWorkspace(workspaceID string) ([]WorkspaceTransaction, error) {
	var tns []WorkspaceTransaction
	tx := s.Orm.Model(&WorkspaceTransaction{}).Where("workspace_id = ?", workspaceID).Find(&tns)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tns, nil

}

func (s *Database) DeleteWorkspaceTransaction(workspaceID string, transactionID string) error {
	return s.Orm.Unscoped().Model(&WorkspaceTransaction{}).
		Where("workspace_id = ? AND transaction_id = ?", workspaceID, transactionID).
		Delete(&WorkspaceTransaction{}).
		Error
}
