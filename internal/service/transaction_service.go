package service

import (
	"context"
	"errors"

	"github.com/alligatorO15/fin-tracker/internal/market"
	"github.com/alligatorO15/fin-tracker/internal/models"
	"github.com/alligatorO15/fin-tracker/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrInvalidTransactionType = errors.New("invalid transaction type")
	ErrTransferMissingAccount = errors.New("transfer requires destination account")
)

type TransactionService interface {
	Create(ctx context.Context, userID uuid.UUID, input *models.TransactionCreate) (*models.Transaction, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error)
	GetByFilter(ctx context.Context, userID uuid.UUID, filter *models.TransactionFilter) (*models.TransactionList, error)
	Update(ctx context.Context, id uuid.UUID, update *models.TransactionUpdate) (*models.Transaction, error)
	Delete(cxt context.Context, id uuid.UUID) error
}

type transactionService struct {
	txManager       repository.TxManager
	transactionRepo repository.TransactionRepository
	accountRepo     repository.AccountRepository
	marketProvider  *market.MultiProvider
}

func NewTransactionService(txManager repository.TxManager, transactionRepo repository.TransactionRepository, accountRepo repository.AccountRepository, marketProvider *market.MultiProvider) TransactionService {
	return &transactionService{
		txManager:       txManager,
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		marketProvider:  marketProvider,
	}
}

func (s *transactionService) Create(ctx context.Context, userID uuid.UUID, input *models.TransactionCreate) (*models.Transaction, error) {
	if input.Type == models.TransactionTypeTransfer {
		if input.ToAccountID == nil {
			return nil, ErrTransferMissingAccount
		}
	}

	// находим счет, чтобы узнать валюту счета
	account, err := s.accountRepo.GetByID(ctx, input.AccountID)
	if err != nil {
		return nil, err
	}

	tx := &models.Transaction{
		UserID:         userID,
		AccountID:      input.AccountID,
		CategoryID:     input.CategoryID,
		Type:           input.Type,
		Amount:         input.Amount,
		Currency:       account.Currency,
		Description:    input.Description,
		Date:           input.Date,
		ToAccountID:    input.ToAccountID,
		ToAmount:       input.ToAmount, // для переводов будет пересчитано ниже с учётом конвертации
		IsRecurring:    input.IsRecurring,
		RecurrenceRule: input.RecurrenceRule,
		Tags:           input.Tags,
		Location:       input.Location,
		Notes:          input.Notes,
	}

	// вычисляем ToAmount для переводов(если нужна конвертация)
	if input.Type == models.TransactionTypeTransfer && input.ToAccountID != nil {
		toAmount := input.Amount
		if input.ToAmount != nil {
			// клиент явно указал сумму (уже сконвертированную)
			toAmount = *input.ToAmount
		} else {
			// конвертируем, если валюта счетов разная
			toAccount, err := s.accountRepo.GetByID(ctx, *input.ToAccountID)
			if err == nil && toAccount.Currency != account.Currency {
				rate, err := s.marketProvider.GetCurrencyRate(ctx, account.Currency, toAccount.Currency)
				if err == nil && !rate.IsZero() {
					toAmount = input.Amount.Mul(rate)
				}

			}
		}
		tx.ToAmount = &toAmount
	}
	// выполняем транзакцию(все репо-методы атомарно)
	err = s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.transactionRepo.Create(txCtx, tx); err != nil {
			return err
		}

		// меняем баланс счета
		switch input.Type {
		case models.TransactionTypeIncome:
			return s.accountRepo.UpdateBalance(txCtx, input.AccountID, input.Amount)
		case models.TransactionTypeExpense:
			return s.accountRepo.UpdateBalance(txCtx, input.AccountID, input.Amount.Neg())
		case models.TransactionTypeTransfer:
			if err := s.accountRepo.UpdateBalance(txCtx, input.AccountID, input.Amount.Neg()); err != nil {
				return err
			}

			if tx.ToAmount != nil {
				return s.accountRepo.UpdateBalance(txCtx, *input.ToAccountID, *tx.ToAmount)
			}
		}
		return nil

	})
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *transactionService) GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	return s.transactionRepo.GetByID(ctx, id)
}

func (s *transactionService) GetByFilter(ctx context.Context, userID uuid.UUID, filter *models.TransactionFilter) (*models.TransactionList, error) {
	return s.transactionRepo.GetByFilter(ctx, userID, filter)
}

func (s *transactionService) Update(ctx context.Context, id uuid.UUID, update *models.TransactionUpdate) (*models.Transaction, error) {
	// Get original transaction
	original, err := s.transactionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var updated *models.Transaction

	err = s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// отменяем изменения на счетах предыдущей старой транзакции
		switch original.Type {
		case models.TransactionTypeIncome:
			if err := s.accountRepo.UpdateBalance(txCtx, original.AccountID, original.Amount.Neg()); err != nil {
				return err
			}
		case models.TransactionTypeExpense:
			if err := s.accountRepo.UpdateBalance(txCtx, original.AccountID, original.Amount); err != nil {
				return err
			}
		case models.TransactionTypeTransfer:
			if err := s.accountRepo.UpdateBalance(txCtx, original.AccountID, original.Amount); err != nil {
				return err
			}
			if original.ToAccountID != nil {
				toAmount := original.Amount
				if original.ToAmount != nil {
					toAmount = *original.ToAmount
				}
				if err := s.accountRepo.UpdateBalance(txCtx, *original.ToAccountID, toAmount.Neg()); err != nil {
					return err
				}
			}
		}

		// апдейтим транзакцию
		if err := s.transactionRepo.Update(txCtx, id, update); err != nil {
			return err
		}

		// получаем новую версию
		var err error
		updated, err = s.transactionRepo.GetByID(txCtx, id)
		if err != nil {
			return err
		}

		// добавляем новые изменения на счетах в соответствии с новой транзакцией
		switch updated.Type {
		case models.TransactionTypeIncome:
			return s.accountRepo.UpdateBalance(txCtx, updated.AccountID, updated.Amount)
		case models.TransactionTypeExpense:
			return s.accountRepo.UpdateBalance(txCtx, updated.AccountID, updated.Amount.Neg())
		case models.TransactionTypeTransfer:
			if err := s.accountRepo.UpdateBalance(txCtx, updated.AccountID, updated.Amount.Neg()); err != nil {
				return err
			}
			if updated.ToAccountID != nil {
				toAmount := updated.Amount
				if updated.ToAmount != nil {
					toAmount = *updated.ToAmount
				}
				return s.accountRepo.UpdateBalance(txCtx, *updated.ToAccountID, toAmount)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *transactionService) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := s.transactionRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context) error {

		switch tx.Type {
		case models.TransactionTypeIncome:
			if err := s.accountRepo.UpdateBalance(txCtx, tx.AccountID, tx.Amount.Neg()); err != nil {
				return err
			}
		case models.TransactionTypeExpense:
			if err := s.accountRepo.UpdateBalance(txCtx, tx.AccountID, tx.Amount); err != nil {
				return err
			}
		case models.TransactionTypeTransfer:
			if err := s.accountRepo.UpdateBalance(txCtx, tx.AccountID, tx.Amount); err != nil {
				return err
			}
			if tx.ToAccountID != nil {
				toAmount := tx.Amount
				if tx.ToAmount != nil {
					toAmount = *tx.ToAmount
				}
				if err := s.accountRepo.UpdateBalance(txCtx, *tx.ToAccountID, toAmount.Neg()); err != nil {
					return err
				}
			}
		}

		return s.transactionRepo.Delete(txCtx, id)
	})
}
