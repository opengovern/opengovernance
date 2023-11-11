package db

func (s *Database) GetCredentialByID(id uint) (*Credential, error) {
	var cred Credential
	err := s.orm.Model(&Credential{}).
		Where("id = ?", id).
		Find(&cred).Error
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

func (s *Database) ListCredentialsByWorkspace(id string) ([]Credential, error) {
	var creds []Credential
	err := s.orm.Model(&Credential{}).
		Where("workspace = ?", id).
		Find(&creds).Error
	if err != nil {
		return nil, err
	}
	return creds, nil
}

func (s *Database) CreateCredential(cred *Credential) error {
	err := s.orm.Model(&Credential{}).
		Create(cred).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Database) SetCredentialCreated(id uint) error {
	err := s.orm.Model(&Credential{}).Where("id = ?", id).Update("is_created", true).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Database) DeleteCredential(id uint) error {
	tx := s.orm.
		Where("id = ?", id).
		Unscoped().
		Delete(&Credential{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
