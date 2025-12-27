# Toggle Deployment Plan

**Recommended Stack**: Vercel + Fly.io + Supabase

This guide walks you through deploying your full-stack application with:
- **Frontend (Next.js)**: Vercel
- **Backend (Go)**: Fly.io
- **Database (PostgreSQL)**: Supabase

---

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Database Setup (Supabase)](#1-database-setup-supabase)
3. [Backend Deployment (Fly.io)](#2-backend-deployment-flyio)
4. [Frontend Deployment (Vercel)](#3-frontend-deployment-vercel)
5. [Environment Variables](#4-environment-variables)
6. [Domain Configuration](#5-domain-configuration)
7. [Monitoring & Maintenance](#6-monitoring--maintenance)
8. [Cost Breakdown](#7-cost-breakdown)
9. [Troubleshooting](#8-troubleshooting)

---

## Prerequisites

### Required Accounts
- [ ] GitHub account (for code repository)
- [ ] Vercel account (sign up at vercel.com)
- [ ] Fly.io account (sign up at fly.io)
- [ ] Supabase account (sign up at supabase.com)

### Required Tools
```bash
# Install Vercel CLI
npm install -g vercel

# Install Fly.io CLI
curl -L https://fly.io/install.sh | sh

# Verify installations
vercel --version
flyctl version
```

### Required Environment Setup
- [ ] Code pushed to GitHub repository
- [ ] Domain name (optional, can use provided subdomains)
- [ ] Resend API key for emails

---

## 1. Database Setup (Supabase)

### Step 1.1: Create Supabase Project

1. Go to https://supabase.com/dashboard
2. Click "New Project"
3. Fill in details:
   - **Name**: toggle-production
   - **Database Password**: (generate strong password, save it!)
   - **Region**: Choose closest to your users (e.g., `us-east-1`)
4. Click "Create new project" (takes ~2 minutes)

### Step 1.2: Get Connection String

1. In Supabase dashboard, go to **Settings** → **Database**
2. Find **Connection String** section
3. Copy the **URI** format:
   ```
   postgresql://postgres:[YOUR-PASSWORD]@db.[PROJECT-REF].supabase.co:5432/postgres
   ```
4. Replace `[YOUR-PASSWORD]` with your actual password
5. Save this - you'll need it for backend and frontend

### Step 1.3: Run Database Migrations

If you have migration files:
```bash
# Install migration tool (if needed)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
export DATABASE_URL="postgresql://postgres:[PASSWORD]@db.[PROJECT-REF].supabase.co:5432/postgres"
migrate -path ./backend/migrations -database "$DATABASE_URL" up
```

Or run SQL directly in Supabase SQL Editor:
1. Go to **SQL Editor** in Supabase dashboard
2. Paste your schema SQL
3. Click "Run"

### Step 1.4: Configure Security

1. Go to **Authentication** → **Policies**
2. Ensure RLS (Row Level Security) is configured if needed
3. Go to **Settings** → **API** and note your:
   - `anon` key (public)
   - `service_role` key (secret, for backend)

---

## 2. Backend Deployment (Fly.io)

### Step 2.1: Initialize Fly App

```bash
# Navigate to backend directory
cd /path/to/toggle/backend

# Login to Fly.io
flyctl auth login

# Launch new app (follow prompts)
flyctl launch

# Choose:
# - App name: toggle-backend (or your preference)
# - Region: Same as Supabase (e.g., iad for us-east-1)
# - Don't create Postgres (we're using Supabase)
# - Don't deploy yet
```

### Step 2.2: Create fly.toml Configuration

Create `backend/fly.toml`:
```toml
app = "toggle-backend"
primary_region = "iad"  # Change to your region

[build]
  builder = "paketobuildpacks/builder:base"

[env]
  PORT = "8080"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 0
  processes = ["app"]

[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 256
```

### Step 2.3: Set Environment Variables

```bash
# Set secrets (not committed to git)
flyctl secrets set \
  DATABASE_URL="postgresql://postgres:[PASSWORD]@db.[PROJECT-REF].supabase.co:5432/postgres" \
  JWT_SECRET="$(openssl rand -base64 32)" \
  CORS_ORIGINS="https://your-app.vercel.app"

# Verify secrets are set
flyctl secrets list
```

### Step 2.4: Configure Backend CORS

Update your Go backend to allow frontend origin:

```go
// backend/main.go or middleware file
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get from env or config
        allowedOrigins := []string{
            os.Getenv("CORS_ORIGINS"),
            "http://localhost:3000", // for local dev
        }

        origin := r.Header.Get("Origin")
        for _, allowed := range allowedOrigins {
            if origin == allowed {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                break
            }
        }

        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        w.Header().Set("Access-Control-Allow-Credentials", "true")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### Step 2.5: Add Health Check Endpoint

```go
// backend/main.go
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
        "version": "1.0.0",
    })
})
```

### Step 2.6: Deploy Backend

```bash
# Deploy to Fly.io
flyctl deploy

# Watch logs
flyctl logs

# Check status
flyctl status

# Test health endpoint
curl https://toggle-backend.fly.dev/health
```

Your backend will be available at: `https://toggle-backend.fly.dev`

---

## 3. Frontend Deployment (Vercel)

### Step 3.1: Prepare Frontend for Deployment

Update `frontend/toggle/next.config.ts`:
```typescript
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactCompiler: true,

  // Add output for optimal performance
  output: 'standalone',

  // Add security headers
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
          {
            key: 'Referrer-Policy',
            value: 'origin-when-cross-origin',
          },
        ],
      },
    ];
  },
};

export default nextConfig;
```

### Step 3.2: Create vercel.json

Create `frontend/toggle/vercel.json`:
```json
{
  "buildCommand": "npm run build",
  "devCommand": "npm run dev",
  "installCommand": "npm install",
  "framework": "nextjs",
  "regions": ["iad1"],
  "env": {
    "NEXT_PUBLIC_API_URL": "https://toggle-backend.fly.dev/api/v1"
  }
}
```

### Step 3.3: Deploy to Vercel

**Option A: Via CLI**
```bash
# Navigate to frontend directory
cd /path/to/toggle/frontend/toggle

# Login to Vercel
vercel login

# Deploy to production
vercel --prod

# Follow prompts:
# - Set up and deploy? Yes
# - Which scope? (Choose your account)
# - Link to existing project? No
# - Project name? toggle
# - Directory? ./
# - Override settings? No
```

**Option B: Via GitHub (Recommended)**
1. Push code to GitHub
2. Go to https://vercel.com/new
3. Import your GitHub repository
4. Configure:
   - **Framework Preset**: Next.js
   - **Root Directory**: `frontend/toggle`
   - **Build Command**: `npm run build`
   - **Output Directory**: `.next`
5. Click "Deploy"

### Step 3.4: Set Environment Variables in Vercel

1. Go to Vercel Dashboard → Your Project → Settings → Environment Variables
2. Add the following:

```bash
# Production Environment Variables
NEXT_PUBLIC_API_URL=https://toggle-backend.fly.dev/api/v1
DATABASE_URL=postgresql://postgres:[PASSWORD]@db.[PROJECT-REF].supabase.co:5432/postgres
BETTER_AUTH_SECRET=your-secret-key-min-32-chars
BETTER_AUTH_URL=https://your-app.vercel.app
RESEND_API_KEY=re_your_resend_api_key

# Google OAuth (if using)
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
```

3. Click "Save"
4. Redeploy: `vercel --prod` or trigger via GitHub push

### Step 3.5: Update Backend CORS

Now that you have your Vercel URL, update backend CORS:
```bash
flyctl secrets set CORS_ORIGINS="https://your-app.vercel.app"
```

---

## 4. Environment Variables

### Complete Environment Variable List

#### Frontend (Vercel)
```bash
# API Configuration
NEXT_PUBLIC_API_URL=https://toggle-backend.fly.dev/api/v1

# Database (for Better Auth)
DATABASE_URL=postgresql://postgres:[PASSWORD]@db.[PROJECT-REF].supabase.co:5432/postgres

# Better Auth
BETTER_AUTH_SECRET=<generate-with-openssl-rand-base64-32>
BETTER_AUTH_URL=https://your-app.vercel.app

# Email Service (Resend)
RESEND_API_KEY=re_xxxxxxxxxxxxx

# OAuth (Optional)
GOOGLE_CLIENT_ID=xxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxx
```

#### Backend (Fly.io)
```bash
# Database
DATABASE_URL=postgresql://postgres:[PASSWORD]@db.[PROJECT-REF].supabase.co:5432/postgres

# Authentication
JWT_SECRET=<generate-with-openssl-rand-base64-32>

# CORS
CORS_ORIGINS=https://your-app.vercel.app

# Server
PORT=8080
```

### Generate Secrets
```bash
# Generate BETTER_AUTH_SECRET
openssl rand -base64 32

# Generate JWT_SECRET
openssl rand -base64 32
```

---

## 5. Domain Configuration

### Option A: Use Provided Domains (Free)
- Frontend: `https://toggle-xxx.vercel.app`
- Backend: `https://toggle-backend.fly.dev`

### Option B: Custom Domain (Recommended)

#### Frontend Custom Domain (Vercel)
1. Buy domain from Namecheap, Cloudflare, etc.
2. In Vercel Dashboard → Settings → Domains
3. Add domain: `yourdomain.com` and `www.yourdomain.com`
4. Update DNS records as instructed by Vercel
5. SSL certificate auto-generated

#### Backend Custom Domain (Fly.io)
1. In your DNS provider, add CNAME:
   ```
   api.yourdomain.com → toggle-backend.fly.dev
   ```
2. Add certificate to Fly.io:
   ```bash
   flyctl certs add api.yourdomain.com
   ```
3. Update frontend environment variable:
   ```bash
   NEXT_PUBLIC_API_URL=https://api.yourdomain.com/api/v1
   ```

---

## 6. Monitoring & Maintenance

### Health Checks

**Backend Health Check**:
```bash
curl https://toggle-backend.fly.dev/health
```

**Frontend Health Check**:
```bash
curl https://your-app.vercel.app/api/health
```

### Logging

**Fly.io Logs**:
```bash
# Real-time logs
flyctl logs

# Follow logs
flyctl logs -f

# Filter by severity
flyctl logs --level error
```

**Vercel Logs**:
1. Go to Vercel Dashboard → Your Project → Logs
2. Or use CLI:
   ```bash
   vercel logs
   ```

**Supabase Logs**:
1. Go to Supabase Dashboard → Logs
2. View query performance, errors, etc.

### Monitoring Setup

**Add Sentry** (Optional but recommended):
```bash
npm install @sentry/nextjs @sentry/node

# Frontend: sentry.client.config.ts
# Backend: Initialize Sentry in main.go
```

**Vercel Analytics**:
1. Go to Vercel Dashboard → Your Project → Analytics
2. Enable Web Analytics (free)

**Fly.io Metrics**:
```bash
flyctl dashboard metrics
```

### Database Backups

Supabase automatically backs up your database:
1. Go to Supabase Dashboard → Database → Backups
2. Daily automatic backups (retained 7 days on free tier)
3. Can manually download backup anytime

### Scaling

**Frontend (Vercel)**:
- Auto-scales automatically
- No configuration needed

**Backend (Fly.io)**:
```bash
# Scale to 2 machines
flyctl scale count 2

# Scale memory
flyctl scale memory 512

# Scale CPU
flyctl scale vm shared-cpu-2x
```

**Database (Supabase)**:
- Free tier: 500MB
- Pro tier: 8GB+ (upgrade in dashboard)

---

## 7. Cost Breakdown

### Free Tier (Development)
- **Vercel**: Free (Hobby plan)
  - Unlimited deployments
  - 100GB bandwidth/month
  - Automatic HTTPS
- **Fly.io**: $0-5/month
  - Free allowances: 3 VMs (256MB each)
  - 160GB outbound data transfer
- **Supabase**: Free
  - 500MB database
  - 1GB file storage
  - 50,000 monthly active users

**Total: $0-5/month**

### Production Tier (Recommended)
- **Vercel Pro**: $20/month
  - 1TB bandwidth
  - Analytics
  - Team features
- **Fly.io**: $5-15/month
  - 1-2 VMs (512MB-1GB each)
  - Autoscaling
- **Supabase Pro**: $25/month
  - 8GB database
  - 100GB file storage
  - 7-day backups

**Total: $50-60/month**

### Enterprise Scale
- **Vercel Enterprise**: Custom pricing
- **Fly.io**: Scale as needed ($30-500+/month)
- **Supabase Team/Enterprise**: $599+/month

---

## 8. Troubleshooting

### Common Issues

#### Backend can't connect to database
```bash
# Check DATABASE_URL is set
flyctl secrets list

# Test connection locally
psql "postgresql://postgres:[PASSWORD]@db.[PROJECT-REF].supabase.co:5432/postgres"

# Check Supabase pooler settings
# Use pooler connection string for better performance
```

#### Frontend gets CORS errors
```bash
# Update backend CORS_ORIGINS
flyctl secrets set CORS_ORIGINS="https://your-app.vercel.app,http://localhost:3000"

# Verify in backend logs
flyctl logs -f
```

#### Authentication not working
```bash
# Check BETTER_AUTH_URL matches your frontend URL
# In Vercel: Settings → Environment Variables
BETTER_AUTH_URL=https://your-app.vercel.app

# Check BETTER_AUTH_SECRET is set and same in backend
# Redeploy frontend after changing
vercel --prod
```

#### Build fails on Vercel
```bash
# Check build logs in Vercel Dashboard
# Common fixes:
# 1. Ensure all dependencies in package.json
# 2. Check TypeScript errors: npm run build locally
# 3. Environment variables set correctly
```

#### Fly.io deployment fails
```bash
# Check builder
# Try different builder in fly.toml:
[build]
  builder = "paketobuildpacks/builder:base"
  # Or use Dockerfile if available

# Check logs
flyctl logs

# Scale up if out of memory
flyctl scale memory 512
```

---

## Deployment Checklist

### Pre-Deployment
- [ ] Code pushed to GitHub
- [ ] Database schema finalized
- [ ] Environment variables documented
- [ ] Health check endpoints added
- [ ] CORS configured correctly
- [ ] Error handling implemented
- [ ] Secrets generated (JWT, auth secrets)

### Database (Supabase)
- [ ] Project created
- [ ] Database password saved securely
- [ ] Connection string obtained
- [ ] Migrations run successfully
- [ ] Test data added (if needed)
- [ ] Backups enabled

### Backend (Fly.io)
- [ ] App created (`flyctl launch`)
- [ ] `fly.toml` configured
- [ ] Secrets set (`flyctl secrets set`)
- [ ] Health check endpoint working
- [ ] CORS configured
- [ ] Deployed successfully
- [ ] Health check URL accessible

### Frontend (Vercel)
- [ ] Project connected to GitHub
- [ ] Environment variables set
- [ ] `NEXT_PUBLIC_API_URL` points to backend
- [ ] `BETTER_AUTH_URL` set correctly
- [ ] Deployed successfully
- [ ] Homepage loads
- [ ] API calls work

### Post-Deployment
- [ ] Custom domains configured (if applicable)
- [ ] SSL certificates verified
- [ ] Test user registration flow
- [ ] Test login flow
- [ ] Test organization creation
- [ ] Monitoring set up
- [ ] Error tracking configured (Sentry)
- [ ] Team notified of URLs

---

## Quick Commands Reference

```bash
# Backend (Fly.io)
flyctl deploy                    # Deploy
flyctl logs -f                   # Watch logs
flyctl ssh console              # SSH into machine
flyctl secrets list             # List secrets
flyctl status                   # Check status
flyctl scale count 2            # Scale to 2 instances

# Frontend (Vercel)
vercel --prod                   # Deploy production
vercel logs                     # View logs
vercel env ls                   # List env vars
vercel domains ls               # List domains

# Database (Supabase)
# Access via dashboard at supabase.com/dashboard
```

---

## Next Steps After Deployment

1. **Set up monitoring**: Add Sentry, Vercel Analytics
2. **Configure CI/CD**: Auto-deploy from GitHub main branch
3. **Add tests**: Integrate testing before deployment
4. **Set up staging**: Create staging environment
5. **Document APIs**: Add API documentation
6. **Performance monitoring**: Set up Lighthouse CI
7. **Security scan**: Run security audit tools

---

## Support Resources

- **Vercel Docs**: https://vercel.com/docs
- **Fly.io Docs**: https://fly.io/docs
- **Supabase Docs**: https://supabase.com/docs
- **Better Auth Docs**: https://better-auth.com/docs

---

**Deployment Plan Version**: 1.0
**Last Updated**: 2025-12-28
